#!/usr/bin/env python3
"""Test SVG printing alignment and scaling by printing to file via the API.

Requires the backend server to be running on localhost:8000.
Generates test labels in debug_output/ and analyzes raster content positions.

Usage:
    python3 scripts/test_svg.py [--base-url http://localhost:8000]
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

# Simple SVG test content: a rectangle with text inside
SVG_RECT = """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 100">
  <rect x="2" y="2" width="196" height="96" fill="none" stroke="black" stroke-width="2"/>
  <text x="100" y="55" text-anchor="middle" font-size="20" font-family="sans-serif">Test SVG</text>
</svg>"""

# Narrow SVG to make alignment differences visible
SVG_NARROW = """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 50 100">
  <rect x="2" y="2" width="46" height="96" fill="black"/>
  <text x="25" y="55" text-anchor="middle" font-size="14" fill="white" font-family="sans-serif">Hi</text>
</svg>"""

# Asymmetric SVG: content on left side only (makes offset easy to spot)
SVG_ASYMMETRIC = """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 100">
  <rect x="5" y="5" width="60" height="90" fill="black"/>
  <text x="35" y="55" text-anchor="middle" font-size="12" fill="white" font-family="sans-serif">L</text>
</svg>"""

H_ALIGNMENTS = ["start", "center", "end"]
V_ALIGNMENTS = ["start", "center", "end"]
LABEL_SIZES = ["62", "62x29"]  # continuous + die-cut
SCALES = [0.5, 1.0, 1.5]


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


def analyze_file(fname: str) -> str:
    """Return a compact analysis string: image size + dark content position."""
    img = Image.open(fname)
    size = f"{img.size[0]}x{img.size[1]}"
    bbox = dark_bbox(fname)
    if bbox is None:
        return f"size={size}  content=EMPTY"
    x0, y0, x1, y1 = bbox
    return f"size={size}  content=x:{x0}-{x1} y:{y0}-{y1}"


def print_svg(svg_data: str, label_size: str, h_align: str = "center",
              v_align: str = "center", scale: float = 1.0,
              custom_height_mm: float = 0, endpoint: str = "/api/print") -> dict:
    """Print an SVG label to file and return dict with filename + raster_file."""
    payload = {
        "svg_data": svg_data,
        "label_size": label_size,
        "printer": "file",
        "model": "QL-820NWB",
        "svg_scale": scale,
        "svg_horizontal_alignment": h_align,
        "svg_vertical_alignment": v_align,
    }
    if custom_height_mm > 0:
        payload["custom_height_mm"] = custom_height_mm
    result = api_post(endpoint, payload)
    if "error" in result:
        raise RuntimeError(f"Print failed: {result['error']}")
    return {"filename": result.get("filename", ""), "raster_file": result.get("raster_file", "")}


def preview_svg(svg_data: str, label_size: str, h_align: str = "center",
                v_align: str = "center", scale: float = 1.0,
                custom_height_mm: float = 0) -> dict:
    """Get a preview for comparison. Returns width, height, and image data."""
    payload = {
        "svg_data": svg_data,
        "label_size": label_size,
        "svg_scale": scale,
        "svg_horizontal_alignment": h_align,
        "svg_vertical_alignment": v_align,
    }
    if custom_height_mm > 0:
        payload["custom_height_mm"] = custom_height_mm
    return api_post("/api/preview", payload)


def test_horizontal_alignment() -> list[str]:
    """Test horizontal alignment with an asymmetric SVG."""
    errors = []
    print("\n--- SVG Horizontal Alignment Test (asymmetric content) ---")

    for label_size in LABEL_SIZES:
        label_desc = f"continuous {label_size}mm" if "x" not in label_size else f"die-cut {label_size}"
        print(f"\n  Label: {label_desc}")

        results = {}
        raster_results = {}
        for h_align in H_ALIGNMENTS:
            time.sleep(1.1)
            try:
                out = print_svg(SVG_ASYMMETRIC, label_size, h_align=h_align)
                info = analyze_file(out["filename"])
                bbox = dark_bbox(out["filename"])
                results[h_align] = bbox
                line = f"    h_align={h_align:6s}  preview: {info}"
                if out["raster_file"] and os.path.exists(out["raster_file"]):
                    r_info = analyze_file(out["raster_file"])
                    r_bbox = dark_bbox(out["raster_file"])
                    raster_results[h_align] = r_bbox
                    line += f"\n    {'':14s}  raster:  {r_info}"
                print(line)
            except Exception as e:
                errors.append(f"[h-align {label_desc}] {h_align}: {e}")
                print(f"    h_align={h_align:6s}  FAILED: {e}")

        # Verify preview: start x < center x < end x
        if len(results) == 3 and all(results.values()):
            xs = {k: v[0] for k, v in results.items()}
            if xs["start"] < xs["center"] < xs["end"]:
                print(f"    PREVIEW OK: x increases (start={xs['start']} < center={xs['center']} < end={xs['end']})")
            elif xs["start"] == xs["center"] == xs["end"]:
                errors.append(f"[h-align {label_desc}] Preview: all identical (x={xs['start']})")
                print(f"    PREVIEW FAIL: All x identical ({xs['start']})")
            else:
                errors.append(f"[h-align {label_desc}] Preview: unexpected x order: {xs}")
                print(f"    PREVIEW WARN: Unexpected x order: {xs}")

        # Verify raster: alignment should also differ in the flipped raster
        if len(raster_results) == 3 and all(raster_results.values()):
            rxs = {k: v[0] for k, v in raster_results.items()}
            # After flip, start (left) becomes right, so start x > center x > end x
            if rxs["start"] > rxs["center"] > rxs["end"]:
                print(f"    RASTER OK: x decreases after flip (start={rxs['start']} > center={rxs['center']} > end={rxs['end']})")
            elif rxs["start"] == rxs["center"] == rxs["end"]:
                errors.append(f"[h-align {label_desc}] Raster: all identical (x={rxs['start']})")
                print(f"    RASTER FAIL: All x identical ({rxs['start']})")
            else:
                print(f"    RASTER INFO: x positions: {rxs}")

    return errors


def test_vertical_alignment() -> list[str]:
    """Test vertical alignment on die-cut (fixed height) labels."""
    errors = []
    label_size = "62x29"
    print(f"\n--- SVG Vertical Alignment Test (die-cut {label_size}) ---")

    results = {}
    for v_align in V_ALIGNMENTS:
        time.sleep(1.1)
        try:
            out = print_svg(SVG_RECT, label_size, v_align=v_align)
            info = analyze_file(out["filename"])
            bbox = dark_bbox(out["filename"])
            results[v_align] = bbox
            line = f"    v_align={v_align:6s}  preview: {info}"
            if out["raster_file"] and os.path.exists(out["raster_file"]):
                r_info = analyze_file(out["raster_file"])
                line += f"\n    {'':14s}  raster:  {r_info}"
            print(line)
        except Exception as e:
            errors.append(f"[v-align] {v_align}: {e}")
            print(f"    v_align={v_align:6s}  FAILED: {e}")

    # Verify: start y < center y < end y
    if len(results) == 3 and all(results.values()):
        ys = {k: v[1] for k, v in results.items()}
        if ys["start"] < ys["center"] < ys["end"]:
            print(f"    OK: y positions increase (start={ys['start']} < center={ys['center']} < end={ys['end']})")
        elif ys["start"] == ys["center"] == ys["end"]:
            errors.append(f"[v-align] All alignments identical (y={ys['start']})")
            print(f"    FAIL: All y positions identical ({ys['start']})")
        else:
            errors.append(f"[v-align] Unexpected y order: {ys}")
            print(f"    WARN: Unexpected y order: {ys}")

    return errors


def test_scaling() -> list[str]:
    """Test that different scales produce different sized content."""
    errors = []
    label_size = "62"
    print(f"\n--- SVG Scale Test (continuous {label_size}mm) ---")

    results = {}
    for scale in SCALES:
        time.sleep(1.1)
        try:
            out = print_svg(SVG_RECT, label_size, scale=scale)
            info = analyze_file(out["filename"])
            bbox = dark_bbox(out["filename"])
            results[scale] = bbox
            print(f"    scale={scale:.1f}  {info}")
        except Exception as e:
            errors.append(f"[scale] {scale}: {e}")
            print(f"    scale={scale:.1f}  FAILED: {e}")

    if len(results) >= 2 and all(results.values()):
        # Content width should grow with scale
        widths = {k: v[2] - v[0] for k, v in results.items()}
        sorted_w = sorted(widths.items())
        if sorted_w[0][1] < sorted_w[-1][1]:
            print(f"    OK: Content width grows ({sorted_w[0][1]}px -> {sorted_w[-1][1]}px)")
        else:
            errors.append("[scale] Content width did not increase with scale")
            print(f"    FAIL: Content width didn't grow: {widths}")

    return errors


def test_preview_vs_print() -> list[str]:
    """Compare preview and print dimensions to detect pipeline differences."""
    errors = []
    print("\n--- SVG Preview vs Print Comparison ---")

    for label_size in LABEL_SIZES:
        label_desc = f"continuous {label_size}mm" if "x" not in label_size else f"die-cut {label_size}"
        print(f"\n  Label: {label_desc}")

        for h_align in H_ALIGNMENTS:
            time.sleep(1.1)
            try:
                prev = preview_svg(SVG_RECT, label_size, h_align=h_align)
                prev_w = prev.get("width", 0)
                prev_h = prev.get("height", 0)

                out = print_svg(SVG_RECT, label_size, h_align=h_align)
                img = Image.open(out["filename"])
                file_w, file_h = img.size

                match = "OK" if (prev_w == file_w and prev_h == file_h) else "MISMATCH"
                line = f"    h_align={h_align:6s}  preview={prev_w}x{prev_h}  file={file_w}x{file_h}  {match}"
                if out["raster_file"] and os.path.exists(out["raster_file"]):
                    r_img = Image.open(out["raster_file"])
                    line += f"  raster={r_img.size[0]}x{r_img.size[1]}"
                print(line)
                if match == "MISMATCH":
                    errors.append(f"[preview-vs-print {label_desc}] {h_align}: preview={prev_w}x{prev_h} file={file_w}x{file_h}")
            except Exception as e:
                errors.append(f"[preview-vs-print {label_desc}] {h_align}: {e}")
                print(f"    h_align={h_align:6s}  FAILED: {e}")

    return errors


def test_endpoints_consistency() -> list[str]:
    """Test that /api/print (with svg_data) and /api/print_svg produce identical output."""
    errors = []
    label_size = "62"
    print(f"\n--- Endpoint Consistency: /api/print vs /api/print_svg ---")

    outputs = {}
    for endpoint_name, endpoint in [("/api/print", "/api/print"), ("/api/print_svg", "/api/print_svg")]:
        time.sleep(1.1)
        try:
            out = print_svg(SVG_RECT, label_size, endpoint=endpoint)
            info = analyze_file(out["filename"])
            outputs[endpoint_name] = out
            print(f"    {endpoint_name:20s}  {info}")
        except Exception as e:
            errors.append(f"[endpoint] {endpoint_name}: {e}")
            print(f"    {endpoint_name:20s}  FAILED: {e}")

    if len(outputs) == 2:
        b1 = dark_bbox(outputs["/api/print"]["filename"])
        b2 = dark_bbox(outputs["/api/print_svg"]["filename"])
        if b1 == b2:
            print(f"    OK: Both endpoints produce identical content position")
        else:
            errors.append(f"[endpoint] Content differs: /api/print={b1} vs /api/print_svg={b2}")
            print(f"    FAIL: Content positions differ")

    return errors


def test_custom_height() -> list[str]:
    """Test SVG on continuous tape with custom height."""
    errors = []
    label_size = "62"
    heights = [0, 20, 50]
    print(f"\n--- SVG Custom Height Test (continuous {label_size}mm) ---")

    results = {}
    for h_mm in heights:
        time.sleep(1.1)
        try:
            out = print_svg(SVG_RECT, label_size, custom_height_mm=h_mm)
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
        heights_px = {k: v[1] for k, v in results.items()}
        print(f"    Canvas heights: {heights_px}")

    return errors


def test_raster_tape_centering() -> list[str]:
    """Verify content positioning within the tape area of the raster image.

    The raster image is 720px wide. The on-tape area is pixels 0-695 (after
    horizontal flip). The off-tape area is pixels 696-719. This test checks
    that content center_x in the raster (relative to tape area 0..printable_width-1)
    is the mirror image of the content center_x in the preview.

    Mirror relationship: raster_tape_x = printable_width - 1 - preview_center_x
    (within ±1px rounding tolerance).
    """
    errors = []
    label_size = "62"
    label_desc = f"continuous {label_size}mm"
    print(f"\n--- Raster Tape Centering Test ({label_desc}) ---")
    print("  Verifying content position within tape area vs preview (mirror relationship)")

    for h_align in H_ALIGNMENTS:
        time.sleep(1.1)
        try:
            out = print_svg(SVG_ASYMMETRIC, label_size, h_align=h_align)
            preview_file = out["filename"]
            raster_file = out["raster_file"]

            if not preview_file or not os.path.exists(preview_file):
                errors.append(f"[tape-centering {label_desc}] {h_align}: preview file missing: {preview_file!r}")
                print(f"    h_align={h_align:6s}  FAILED: preview file missing")
                continue
            if not raster_file or not os.path.exists(raster_file):
                errors.append(f"[tape-centering {label_desc}] {h_align}: raster file missing: {raster_file!r}")
                print(f"    h_align={h_align:6s}  FAILED: raster file missing")
                continue

            # Preview image width = printable width (tape area in raster)
            prev_img = Image.open(preview_file)
            printable_width = prev_img.size[0]  # e.g. 696px

            prev_bbox = dark_bbox(preview_file)
            if prev_bbox is None:
                errors.append(f"[tape-centering {label_desc}] {h_align}: preview has no dark content")
                print(f"    h_align={h_align:6s}  FAILED: no dark content in preview")
                continue

            # Content center_x in preview image coordinates
            prev_center_x = (prev_bbox[0] + prev_bbox[2]) / 2.0

            # Raster image: tape area is columns 0..(printable_width-1)
            raster_img = Image.open(raster_file)
            raster_w = raster_img.size[0]  # 720px
            raster_bbox = dark_bbox(raster_file)

            if raster_bbox is None:
                errors.append(f"[tape-centering {label_desc}] {h_align}: raster has no dark content")
                print(f"    h_align={h_align:6s}  FAILED: no dark content in raster")
                continue

            # Content center_x in raster image coordinates (tape area: 0..printable_width-1)
            raster_center_x = (raster_bbox[0] + raster_bbox[2]) / 2.0

            # Expected raster position: mirror of preview within tape area
            expected_raster_x = (printable_width - 1) - prev_center_x

            # Margins within tape area
            raster_left_margin = raster_bbox[0]
            raster_right_margin = (printable_width - 1) - raster_bbox[2]

            diff = abs(raster_center_x - expected_raster_x)
            tolerance = 1  # px

            status = "OK" if diff <= tolerance else "FAIL"
            print(
                f"    h_align={h_align:6s}  "
                f"preview_cx={prev_center_x:.1f}  "
                f"expected_raster_cx={expected_raster_x:.1f}  "
                f"actual_raster_cx={raster_center_x:.1f}  "
                f"diff={diff:.1f}px  "
                f"tape_margins(L/R)={raster_left_margin}/{raster_right_margin}  "
                f"raster_size={raster_w}x{raster_img.size[1]}  "
                f"{status}"
            )
            if diff > tolerance:
                errors.append(
                    f"[tape-centering {label_desc}] {h_align}: "
                    f"raster_cx={raster_center_x:.1f} expected={expected_raster_x:.1f} "
                    f"(diff={diff:.1f}px > {tolerance}px tolerance)"
                )
        except Exception as e:
            errors.append(f"[tape-centering {label_desc}] {h_align}: {e}")
            print(f"    h_align={h_align:6s}  FAILED: {e}")

    return errors


def main():
    parser = argparse.ArgumentParser(description="Test SVG label printing alignment and scaling")
    parser.add_argument("--base-url", default=CONFIG["base_url"], help="Backend server URL")
    args = parser.parse_args()
    CONFIG["base_url"] = args.base_url

    try:
        api_get("/api/config")
    except urllib.error.URLError:
        sys.exit(f"Cannot connect to server at {CONFIG['base_url']}. Is the backend running?")

    os.makedirs(OUTPUT_DIR, exist_ok=True)

    all_errors = []
    all_errors.extend(test_horizontal_alignment())
    all_errors.extend(test_vertical_alignment())
    all_errors.extend(test_scaling())
    all_errors.extend(test_preview_vs_print())
    all_errors.extend(test_endpoints_consistency())
    all_errors.extend(test_custom_height())
    all_errors.extend(test_raster_tape_centering())

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
