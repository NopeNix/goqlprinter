#!/usr/bin/env python3
"""Test QR code printing by printing to file via the API.

Requires the backend server to be running on localhost:8000.
Generates test labels in debug_output/ and analyzes raster content positions.

Usage:
    python3 scripts/test_qr.py [--base-url http://localhost:8000]
"""

import argparse
import json
import os
import sys
import time
import urllib.request
import urllib.error

from PIL import Image
import numpy as np


CONFIG = {"base_url": "http://localhost:8000"}
OUTPUT_DIR = "debug_output"

LABEL_SIZES = ["62", "62x29", "29"]  # continuous wide, die-cut, continuous narrow
QR_DATA_SHORT = "https://example.com"
QR_DATA_LONG = "https://example.com/this-is-a-much-longer-url-to-test-qr-density-and-scaling-behavior"


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


def dark_bbox(path: str, threshold: int = 200) -> tuple[int, int, int, int] | None:
    """Find bounding box of dark (non-white) pixels in an image.

    Returns (x_min, y_min, x_max, y_max) or None if no dark pixels found.
    """
    pixels = np.array(Image.open(path).convert("L"))
    dark = np.where(pixels < threshold)
    if len(dark[0]) == 0:
        return None
    return (int(dark[1].min()), int(dark[0].min()), int(dark[1].max()), int(dark[0].max()))


def analyze_centering(path: str) -> str:
    """Analyze how centered the dark content is within the image.

    Returns a string describing centering offset in both axes.
    """
    img = Image.open(path)
    w, h = img.size
    bbox = dark_bbox(path)
    if bbox is None:
        return "EMPTY"
    x0, y0, x1, y1 = bbox
    content_cx = (x0 + x1) / 2
    content_cy = (y0 + y1) / 2
    offset_x = content_cx - w / 2
    offset_y = content_cy - h / 2
    margin_l = x0
    margin_r = w - 1 - x1
    margin_t = y0
    margin_b = h - 1 - y1
    return (
        f"margins L={margin_l} R={margin_r} T={margin_t} B={margin_b}  "
        f"center_offset=({offset_x:+.0f}, {offset_y:+.0f})"
    )


def analyze_file(path: str) -> str:
    """Return a compact analysis string: image size + dark content position."""
    img = Image.open(path)
    size = f"{img.size[0]}x{img.size[1]}"
    bbox = dark_bbox(path)
    if bbox is None:
        return f"size={size}  content=EMPTY"
    x0, y0, x1, y1 = bbox
    return f"size={size}  content=x:{x0}-{x1} y:{y0}-{y1}"


def print_qr(data: str, label_size: str, custom_height_mm: float = 0) -> dict:
    """Print a QR code label to file and return dict with filename + raster_file."""
    payload = {
        "data": data,
        "label_size": label_size,
        "printer": "file",
        "model": "QL-820NWB",
    }
    if custom_height_mm > 0:
        payload["custom_height_mm"] = custom_height_mm
    result = api_post("/api/print_qr", payload)
    if "error" in result:
        raise RuntimeError(f"Print failed: {result['error']}")
    return {"filename": result.get("filename", ""), "raster_file": result.get("raster_file", "")}


def test_label_sizes() -> list[str]:
    """Test QR code generation on different label sizes."""
    errors = []
    print("\n--- QR Label Size Test ---")

    for label_size in LABEL_SIZES:
        time.sleep(1.1)
        label_desc = f"continuous {label_size}mm" if "x" not in label_size else f"die-cut {label_size}"
        try:
            out = print_qr(QR_DATA_SHORT, label_size)
            info = analyze_file(out["filename"])
            print(f"    {label_desc:25s}  {info}")
        except Exception as e:
            errors.append(f"[label-size] {label_desc}: {e}")
            print(f"    {label_desc:25s}  FAILED: {e}")

    return errors


def test_data_density() -> list[str]:
    """Test short vs long QR data to verify density differences."""
    errors = []
    label_size = "62"
    print(f"\n--- QR Data Density Test (continuous {label_size}mm) ---")

    results = {}
    for desc, data in [("short", QR_DATA_SHORT), ("long", QR_DATA_LONG)]:
        time.sleep(1.1)
        try:
            out = print_qr(data, label_size)
            info = analyze_file(out["filename"])
            bbox = dark_bbox(out["filename"])
            results[desc] = bbox
            print(f"    data={desc:5s} ({len(data):3d} chars)  {info}")
        except Exception as e:
            errors.append(f"[density] {desc}: {e}")
            print(f"    data={desc:5s}  FAILED: {e}")

    if len(results) == 2 and all(results.values()):
        w_short = results["short"][2] - results["short"][0]
        w_long = results["long"][2] - results["long"][0]
        print(f"    OK: Content widths short={w_short}px long={w_long}px")

    return errors


def test_custom_height() -> list[str]:
    """Test QR code with custom height on continuous tape."""
    errors = []
    label_size = "62"
    heights = [0, 30, 60]
    print(f"\n--- QR Custom Height Test (continuous {label_size}mm) ---")

    results = {}
    for h_mm in heights:
        time.sleep(1.1)
        try:
            out = print_qr(QR_DATA_SHORT, label_size, custom_height_mm=h_mm)
            info = analyze_file(out["filename"])
            img = Image.open(out["filename"])
            results[h_mm] = img.size
            desc = "auto" if h_mm == 0 else f"{h_mm}mm"
            print(f"    height={desc:6s}  {info}")
        except Exception as e:
            desc = "auto" if h_mm == 0 else f"{h_mm}mm"
            errors.append(f"[custom-height] {desc}: {e}")
            print(f"    height={desc:6s}  FAILED: {e}")

    if len(results) >= 2:
        heights_px = {(f"{k}mm" if k else "auto"): v[1] for k, v in results.items()}
        print(f"    Canvas heights: {heights_px}")

    return errors


def test_minimum_size() -> list[str]:
    """Test QR code on smallest label to verify minimum size check."""
    errors = []
    label_size = "12"  # 12mm tape - may be too small
    print(f"\n--- QR Minimum Size Test ({label_size}mm tape) ---")

    time.sleep(1.1)
    try:
        out = print_qr(QR_DATA_SHORT, label_size)
        info = analyze_file(out["filename"])
        print(f"    {label_size}mm  {info}")
    except RuntimeError as e:
        if "too small" in str(e).lower():
            print(f"    {label_size}mm  Correctly rejected: {e}")
        else:
            errors.append(f"[min-size] Unexpected error: {e}")
            print(f"    {label_size}mm  FAILED: {e}")
    except Exception as e:
        errors.append(f"[min-size] {e}")
        print(f"    {label_size}mm  FAILED: {e}")

    return errors


def test_centering() -> list[str]:
    """Test QR code centering on different label types.

    QR code should be centered (equal margins left/right and top/bottom).
    """
    errors = []
    print("\n--- QR Centering Test ---")

    for label_size in ["62", "62x29"]:
        time.sleep(1.1)
        label_desc = f"continuous {label_size}mm" if "x" not in label_size else f"die-cut {label_size}"
        try:
            out = print_qr(QR_DATA_SHORT, label_size)
            info = analyze_file(out["filename"])
            centering = analyze_centering(out["filename"])
            print(f"    {label_desc:25s}  preview: {info}")
            print(f"    {' ':25s}  {centering}")

            bbox = dark_bbox(out["filename"])
            if bbox:
                img = Image.open(out["filename"])
                w, h = img.size
                x0, y0, x1, y1 = bbox
                margin_l, margin_r = x0, w - 1 - x1
                margin_diff = abs(margin_l - margin_r)
                if margin_diff > 2:
                    errors.append(f"[centering {label_desc}] H-margins uneven: L={margin_l} R={margin_r} (diff={margin_diff})")
                    print(f"    {' ':25s}  WARN: Horizontal centering off by {margin_diff}px")
                else:
                    print(f"    {' ':25s}  OK: Preview centered (margin diff <= 2px)")

            if out["raster_file"] and os.path.exists(out["raster_file"]):
                r_info = analyze_file(out["raster_file"])
                print(f"    {' ':25s}  raster:  {r_info}")

                r_bbox = dark_bbox(out["raster_file"])
                if r_bbox:
                    # Check centering within tape area (0 to preview_width-1),
                    # not the full raster width (which includes off-tape pixels).
                    tape_w = w  # preview width = printable tape width
                    rx0, ry0, rx1, ry1 = r_bbox
                    r_margin_l = rx0
                    r_margin_r = tape_w - 1 - rx1
                    r_margin_diff = abs(r_margin_l - r_margin_r)
                    print(f"    {' ':25s}  tape margins L={r_margin_l} R={r_margin_r} (tape_w={tape_w})")
                    if r_margin_diff > 2:
                        errors.append(f"[centering {label_desc}] Raster tape-margins uneven: L={r_margin_l} R={r_margin_r} (diff={r_margin_diff})")
                        print(f"    {' ':25s}  WARN: Raster tape centering off by {r_margin_diff}px")
                    else:
                        print(f"    {' ':25s}  OK: Raster centered on tape (margin diff <= 2px)")
        except Exception as e:
            errors.append(f"[centering] {label_desc}: {e}")
            print(f"    {label_desc:25s}  FAILED: {e}")

    return errors


def main():
    parser = argparse.ArgumentParser(description="Test QR code label printing")
    parser.add_argument("--base-url", default=CONFIG["base_url"], help="Backend server URL")
    args = parser.parse_args()
    CONFIG["base_url"] = args.base_url

    try:
        api_get("/api/config")
    except urllib.error.URLError:
        sys.exit(f"Cannot connect to server at {CONFIG['base_url']}. Is the backend running?")

    os.makedirs(OUTPUT_DIR, exist_ok=True)

    all_errors = []
    all_errors.extend(test_label_sizes())
    all_errors.extend(test_data_density())
    all_errors.extend(test_custom_height())
    all_errors.extend(test_minimum_size())
    all_errors.extend(test_centering())

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
