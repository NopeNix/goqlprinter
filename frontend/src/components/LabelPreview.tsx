import React, { useEffect, useState } from "react";
import { QRCodeCanvas } from "qrcode.react";

interface LabelPreviewProps {
  labelText: string;
  svgData?: string | null;
  qrData?: string;
  pngData?: File | null;
  selectedFont: string;
  fontSize: number;
  labelWidth: number; // Physical width in mm
  labelHeight: number; // Physical height in mm
  dotsTotalWidth: number;
  dotsTotalHeight: number;
  printableLabelWidth: number;
  printableLabelHeight: number;
  orientation: "standard" | "rotated";
  horizontalAlignment: "start" | "center" | "end";
  verticalAlignment: "start" | "center" | "end";
  textRotation: number;
  svgScale: number;
  svgHorizontalAlignment: "start" | "center" | "end";
  svgVerticalAlignment: "start" | "center" | "end";
  previewUrl?: string | null; // Backend-rendered preview image
  customHeightMM?: number; // Custom height for endless tape in mm
  heightMode?: "auto" | "manual"; // Height mode for endless tape
}

const PREVIEW_SCALE_FACTOR = 4; // Scales mm to pixels for the outer container
const DOTS_PER_MM = 11.81; // 300 DPI / 25.4 mm/inch

const LabelPreview: React.FC<LabelPreviewProps> = ({
  labelText,
  svgData,
  qrData,
  pngData,
  selectedFont,
  fontSize,
  labelWidth,
  labelHeight,
  dotsTotalWidth,
  dotsTotalHeight,
  printableLabelWidth,
  printableLabelHeight,
  orientation,
  horizontalAlignment,
  verticalAlignment,
  textRotation,
  svgScale,
  svgHorizontalAlignment,
  svgVerticalAlignment,
  previewUrl,
  customHeightMM = 0,
  heightMode = "auto",
}) => {
  const [isFontAvailable, setIsFontAvailable] = useState(true);

  useEffect(() => {
    if (selectedFont) {
      // Check if the font is loaded and available for rendering
      document.fonts.ready.then(() => {
        if (document.fonts.check(`12px "${selectedFont}"`)) {
          setIsFontAvailable(true);
        } else {
          setIsFontAvailable(false);
        }
      });
    }
  }, [selectedFont]);

  const isLabelRotated = orientation === 'rotated';

  // Endless tape detection and manual height handling
  const isEndlessTape = labelHeight === 0;
  const useManualHeight = isEndlessTape && heightMode === "manual" && customHeightMM > 0;

  // For endless tape with manual height, compute effective dimensions
  // For auto mode on endless tape, use a default preview height if backend hasn't provided dimensions
  const autoEndlessHeightMM = isEndlessTape && !useManualHeight ? Math.max(labelWidth, 50) : 0;
  const effectiveLabelHeight = useManualHeight
    ? customHeightMM
    : isEndlessTape
      ? autoEndlessHeightMM
      : labelHeight;
  const effectiveDotsTotalHeight = useManualHeight
    ? Math.round(customHeightMM * DOTS_PER_MM)
    : isEndlessTape && !useManualHeight
      ? Math.round(autoEndlessHeightMM * DOTS_PER_MM)
      : dotsTotalHeight;
  const effectivePrintableLabelHeight = useManualHeight
    ? Math.round(customHeightMM * DOTS_PER_MM)
    : isEndlessTape && !useManualHeight
      ? Math.round(autoEndlessHeightMM * DOTS_PER_MM)
      : printableLabelHeight;

  // Use physical dimensions for the outer container to keep preview size consistent
  const containerWidth = (isLabelRotated ? effectiveLabelHeight : labelWidth) * PREVIEW_SCALE_FACTOR;
  const containerHeight = (isLabelRotated ? labelWidth : effectiveLabelHeight) * PREVIEW_SCALE_FACTOR;

  // Calculate the scaling factor from total dots to container pixels
  const scaleX =
    containerWidth / (isLabelRotated ? effectiveDotsTotalHeight : dotsTotalWidth);
  const scaleY =
    containerHeight / (isLabelRotated ? dotsTotalWidth : effectiveDotsTotalHeight);

  // Use the smaller scale factor to maintain aspect ratio
  const scale = Math.min(scaleX, scaleY);

  // Dimensions of the entire label area in pixels
  const totalWidthPx =
    (isLabelRotated ? effectiveDotsTotalHeight : dotsTotalWidth) * scale;
  const totalHeightPx =
    (isLabelRotated ? dotsTotalWidth : effectiveDotsTotalHeight) * scale;

  // Dimensions of the printable area in pixels
  const printableWidthPx =
    (isLabelRotated ? effectivePrintableLabelHeight : printableLabelWidth) * scale;
  const printableHeightPx =
    (isLabelRotated ? printableLabelWidth : effectivePrintableLabelHeight) * scale;

  // Margins in pixels
  const marginX = (totalWidthPx - printableWidthPx) / 2;
  const marginY = (totalHeightPx - printableHeightPx) / 2;

  // Calculate printable area in mm for display
  const printableWidthMm = (printableLabelWidth / DOTS_PER_MM).toFixed(1);
  const displayPrintableHeightMm = useManualHeight
    ? customHeightMM.toFixed(1)
    : (printableLabelHeight / DOTS_PER_MM).toFixed(1);

  const containerStyle: React.CSSProperties = {
    width: `${totalWidthPx}px`,
    height: `${totalHeightPx}px`,
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    border: "1px dashed #aaa",
    backgroundColor: "#f0f0f0",
    padding: `${marginY}px ${marginX}px`,
    boxSizing: "border-box",
  };

  const printableAreaStyle: React.CSSProperties = {
    width: `${printableWidthPx}px`,
    height: `${printableHeightPx}px`,
    backgroundColor: "white",
    border: "1px solid #ccc",
    overflow: "hidden",
    position: "relative",
  };

  const contentWrapperStyle: React.CSSProperties = {
    width: '100%',
    height: '100%',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    transformOrigin: 'center',
  };

  const isTextRotated = textRotation === 90 || textRotation === 270;

  // Calculate flexbox alignment properties based on text rotation
  let flexJustify: React.CSSProperties['justifyContent'] = horizontalAlignment;
  let flexAlign: React.CSSProperties['alignItems'] = verticalAlignment;

  if (textRotation === 90) {
    // For 90° rotation:
    // - Horizontal alignment becomes vertical alignment (but needs inversion)
    // - Vertical alignment becomes horizontal alignment
    flexJustify = verticalAlignment;
    if (horizontalAlignment === 'start') flexAlign = 'end';
    else if (horizontalAlignment === 'end') flexAlign = 'start';
    else flexAlign = 'center';
  } else if (textRotation === 270) {
    // For 270° rotation:
    // - Horizontal alignment becomes vertical alignment
    // - Vertical alignment becomes horizontal alignment (but needs inversion)
    flexAlign = horizontalAlignment;
    if (verticalAlignment === 'start') flexJustify = 'end';
    else if (verticalAlignment === 'end') flexJustify = 'start';
    else flexJustify = 'center';
  }


  const textContainerStyle: React.CSSProperties = {
    width: '100%',
    height: '100%',
    display: 'flex',
    justifyContent: flexJustify,
    alignItems: flexAlign,
  };

  const textWrapperStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    transform: `rotate(${textRotation}deg)`,
    transformOrigin: 'center',
    color: isFontAvailable ? 'black' : 'transparent', // Hide text if font not ready, but keep space
  }

  // Backend uses: scaledFontSize = fontSize * 4 on printableLabelWidth canvas
  // Frontend preview must scale proportionally: fontSize * 4 * scale
  const scaledFontSize = fontSize * 4 * scale;

  const textStyle: React.CSSProperties = {
    fontFamily: isFontAvailable ? selectedFont : 'sans-serif',
    fontSize: `${scaledFontSize}px`,
    lineHeight: 1,
    whiteSpace: 'pre', // Preserve newlines but don't wrap text
    maxWidth: isTextRotated ? `${printableHeightPx}px` : `${printableWidthPx}px`,
    maxHeight: isTextRotated ? `${printableWidthPx}px` : `${printableHeightPx}px`,
  };

  const renderContent = () => {
    // If we have a backend preview, show it (takes priority)
    if (previewUrl) {
      return (
        <div className="flex items-center justify-center w-full h-full">
          <img
            src={previewUrl}
            alt="Preview"
            style={{
              maxWidth: '100%',
              maxHeight: '100%',
              objectFit: 'contain'
            }}
          />
        </div>
      );
    }

    // Fallback to client-side rendering
    if (pngData) {
      return (
        <div className="flex items-center justify-center w-full h-full">
          <img
            src={URL.createObjectURL(pngData)}
            alt="Preview"
            style={{
              maxWidth: '100%',
              maxHeight: '100%',
              objectFit: 'contain'
            }}
          />
        </div>
      );
    }
    if (qrData) {
      const qrSize = Math.min(printableWidthPx, printableHeightPx) * 0.9;
      return (
        <div className="flex items-center justify-center w-full h-full">
          <QRCodeCanvas value={qrData} size={qrSize} />
        </div>
      );
    }
    if (svgData) {
      // Parse SVG viewBox to get intrinsic dimensions
      const viewBoxMatch = svgData.match(/viewBox="([^"]+)"/);
      if (!viewBoxMatch) {
        return null;
      }
      
      const [, , svgWidth, svgHeight] = viewBoxMatch[1].split(/\s+|,/).map(parseFloat);
      
      // Calculate scale factor to fit printable area while maintaining aspect ratio
      const widthScale = printableWidthPx / svgWidth;
      const heightScale = printableHeightPx / svgHeight;
      const baseScale = Math.min(widthScale, heightScale);
      
      // Apply user's additional scale factor
      const finalScale = baseScale * svgScale;
      
      const scaledWidth = svgWidth * finalScale;
      const scaledHeight = svgHeight * finalScale;

      // Calculate alignment offsets
      let left = 0;
      if (svgHorizontalAlignment === 'center') {
        left = (printableWidthPx - scaledWidth) / 2;
      } else if (svgHorizontalAlignment === 'end') {
        left = printableWidthPx - scaledWidth;
      }

      let top = 0;
      if (svgVerticalAlignment === 'center') {
        top = (printableHeightPx - scaledHeight) / 2;
      } else if (svgVerticalAlignment === 'end') {
        top = printableHeightPx - scaledHeight;
      }

      // Create transformed SVG with proper scaling
      const transformedSvg = svgData
        .replace(/width="[^"]*"/, `width="${scaledWidth}"`)
        .replace(/height="[^"]*"/, `height="${scaledHeight}"`)
        .replace(/viewBox="[^"]*"/, `viewBox="0 0 ${svgWidth} ${svgHeight}"`);

      return (
        <div
          style={{
            position: 'absolute',
            left: `${left}px`,
            top: `${top}px`,
            width: `${scaledWidth}px`,
            height: `${scaledHeight}px`,
          }}
          dangerouslySetInnerHTML={{ __html: transformedSvg }}
        />
      );
    }
    return (
      <div style={textContainerStyle}>
        <div style={textWrapperStyle}>
          <div style={textStyle}>{labelText}</div>
        </div>
      </div>
    );
  };

  return (
    <div className="flex flex-col items-center space-y-2">
      {!isFontAvailable && (
        <div className="text-sm text-red-500">
          Preview not available for this font. It will print correctly.
        </div>
      )}
      <div style={containerStyle}>
        <div style={printableAreaStyle}>
          <div style={contentWrapperStyle}>{renderContent()}</div>
        </div>
      </div>
      <div className="text-sm text-gray-500">
        Printable Area:{" "}
        {isLabelRotated
          ? `${displayPrintableHeightMm} x ${printableWidthMm}`
          : `${printableWidthMm} x ${displayPrintableHeightMm}`}{" "}
        mm
        {isEndlessTape && heightMode === "auto" && " (auto)"}
      </div>
    </div>
  );
};

export default LabelPreview;
