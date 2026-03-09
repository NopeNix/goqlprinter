#!/usr/bin/env python3
"""Test text rotation and alignment by printing to file via the API.

Requires the backend server to be running on localhost:8000.
Generates test labels in debug_output/ and verifies they differ.

Usage:
    python3 scripts/test_rotation.py [--base-url http://localhost:8000]
"""

import argparse
import json
import os
import sys
import time
import urllib.request
import urllib.error


CONFIG = {"base_url": "http://localhost:8000"}
OUTPUT_DIR = "debug_output"

# Test matrix
ROTATIONS = [0, 90, 180, 270]
H_ALIGNMENTS = ["start", "center", "end"]
V_ALIGNMENTS = ["start", "center", "end"]
LABEL_SIZES = ["62", "62x29"]  # continuous + die-cut


def api_post(path: str, payload: dict) -> dict:
    url = f"{CONFIG['base_url']}{path}"
    data = json.dumps(payload).encode()
    req = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=10) as resp:
        return json.loads(resp.read())


def api_get(path: str) -> dict:
    url = f"{CONFIG['base_url']}{path}"
    with urllib.request.urlopen(url, timeout=10) as resp:
        return json.loads(resp.read())


def get_first_font() -> str:
    data = api_get("/api/fonts")
    fonts = data.get("fonts", [])
    if not fonts:
        sys.exit("No fonts available on server")
    for preferred in ["Roboto", "Roboto-Regular", "cmunbsr"]:
        if preferred in fonts:
            return preferred
    return fonts[0]


def print_label(text: str, label_size: str, font: str, font_size: int,
                rotation: int, h_align: str, v_align: str) -> str:
    """Print a label to file and return the output filename."""
    payload = {
        "text": text,
        "label_size": label_size,
        "font_family": font,
        "font_size": font_size,
        "printer": "file",
        "model": "QL-820NWB",
        "orientation": "standard",
        "horizontal_alignment": h_align,
        "vertical_alignment": v_align,
        "text_rotation": rotation,
    }
    result = api_post("/api/print", payload)
    if "error" in result:
        raise RuntimeError(f"Print failed: {result['error']}")
    return result.get("filename", "")


def test_rotations(font: str) -> list[str]:
    """Test all 4 rotations with center alignment on both label types."""
    errors = []
    for label_size in LABEL_SIZES:
        fsize = 20 if "x" in label_size else 40
        label_desc = f"continuous {label_size}mm" if "x" not in label_size else f"die-cut {label_size}"
        print(f"\n--- Rotation test: {label_desc} ---")

        files = {}
        for rot in ROTATIONS:
            time.sleep(1.1)
            try:
                fname = print_label("Test", label_size, font, fsize, rot, "center", "center")
                fsize_bytes = os.path.getsize(fname)
                files[rot] = (fname, fsize_bytes)
                print(f"  rot={rot:3d}°  -> {fname} ({fsize_bytes} bytes)")
            except Exception as e:
                errors.append(f"[{label_desc}] rot={rot}: {e}")
                print(f"  rot={rot:3d}°  -> FAILED: {e}")

        if len(files) == 4:
            sizes = {rot: s for rot, (_, s) in files.items()}
            if sizes[0] == sizes[90] and sizes[0] == sizes[180] and sizes[0] == sizes[270]:
                errors.append(f"[{label_desc}] All rotations produced identical file sizes - rotation may not be working")
                print(f"  WARNING: All files same size ({sizes[0]} bytes)")
            else:
                print(f"  OK: File sizes vary across rotations")

    return errors


def test_alignments(font: str) -> list[str]:
    """Test horizontal alignment with 90° rotation on continuous tape."""
    errors = []
    label_size = "62"
    rotation = 90
    print(f"\n--- Alignment test: continuous {label_size}mm, rotation={rotation}° ---")

    files = {}
    for h_align in H_ALIGNMENTS:
        time.sleep(1.1)
        try:
            fname = print_label("Test", label_size, font, 40, rotation, h_align, "center")
            fsize_bytes = os.path.getsize(fname)
            files[h_align] = (fname, fsize_bytes)
            print(f"  h_align={h_align:6s}  -> {fname} ({fsize_bytes} bytes)")
        except Exception as e:
            errors.append(f"[alignment] h_align={h_align}: {e}")
            print(f"  h_align={h_align:6s}  -> FAILED: {e}")

    if len(files) == 3:
        sizes = [s for _, s in files.values()]
        if len(set(sizes)) == 1:
            errors.append("[alignment] All alignments produced identical file sizes - alignment may not be working")
            print(f"  WARNING: All files same size")
        else:
            print(f"  OK: File sizes vary across alignments")

    return errors


def test_vertical_alignment_diecut(font: str) -> list[str]:
    """Test vertical alignment on die-cut label (fixed height allows v-align)."""
    errors = []
    label_size = "62x29"
    print(f"\n--- Vertical alignment test: die-cut {label_size} ---")

    files = {}
    for v_align in V_ALIGNMENTS:
        time.sleep(1.1)
        try:
            fname = print_label("Hi", label_size, font, 15, 0, "center", v_align)
            fsize_bytes = os.path.getsize(fname)
            files[v_align] = (fname, fsize_bytes)
            print(f"  v_align={v_align:6s}  -> {fname} ({fsize_bytes} bytes)")
        except Exception as e:
            errors.append(f"[v-alignment] v_align={v_align}: {e}")
            print(f"  v_align={v_align:6s}  -> FAILED: {e}")

    if len(files) == 3:
        sizes = [s for _, s in files.values()]
        if len(set(sizes)) == 1:
            errors.append("[v-alignment] All vertical alignments produced identical file sizes")
            print(f"  WARNING: All files same size")
        else:
            print(f"  OK: File sizes vary across vertical alignments")

    return errors


def main():
    parser = argparse.ArgumentParser(description="Test label rotation and alignment")
    parser.add_argument("--base-url", default=CONFIG["base_url"], help="Backend server URL")
    args = parser.parse_args()
    CONFIG["base_url"] = args.base_url

    # Check server is running
    try:
        api_get("/api/config")
    except urllib.error.URLError:
        sys.exit(f"Cannot connect to server at {CONFIG['base_url']}. Is the backend running?")

    os.makedirs(OUTPUT_DIR, exist_ok=True)

    font = get_first_font()
    print(f"Using font: {font}")

    all_errors = []
    all_errors.extend(test_rotations(font))
    all_errors.extend(test_alignments(font))
    all_errors.extend(test_vertical_alignment_diecut(font))

    print("\n" + "=" * 50)
    if all_errors:
        print(f"DONE with {len(all_errors)} issue(s):")
        for err in all_errors:
            print(f"  - {err}")
        sys.exit(1)
    else:
        print("ALL TESTS PASSED")


if __name__ == "__main__":
    main()
