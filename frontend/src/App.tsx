import { useState, useCallback, useEffect, useRef } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./components/ui/select";
import ModeToggle from "./components/ModeToggle";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "./components/ui/card";
import { Button } from "./components/ui/button";
import { saveSettings, loadSettings } from "./utils/localStorageUtils";
import { Textarea } from "./components/ui/textarea";
import { Label } from "./components/ui/label";
import { Slider } from "./components/ui/slider";
import PrinterSelector from "./components/PrinterSelector";
import PrinterDisconnectedNotification from "./components/PrinterDisconnectedNotification";
import LabelSizeSelector from "./components/LabelSizeSelector";
import FontSelector from "./components/FontSelector";
import LabelPreview from "./components/LabelPreview";
import QRCodeGenerator from "./components/QRCodeGenerator";
import TextAlignmentSelector from "./components/TextAlignmentSelector";
import usePrinterRecovery, { type PrinterInfo } from "./hooks/usePrinterRecovery";
import { usePreview } from "./hooks/usePreview";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "./components/ui/accordion";
import { RadioGroup, RadioGroupItem } from "./components/ui/radio-group";

type PrintMode = "text" | "qr" | "svg" | "png";

interface AppSettings {
  selectedPrinter: { id: string; name: string } | null;
  selectedLabelSize: string;
  labelWidth: number;
  labelHeight: number;
  dotsTotalWidth: number;
  dotsTotalHeight: number;
  printableLabelWidth: number;
  printableLabelHeight: number;
  selectedOrientation: string;
  selectedFont: string;
  fontSize: number[];
  labelText: string;
  qrData: string;
  printMode: PrintMode;
  horizontalAlignment: "start" | "center" | "end";
  verticalAlignment: "start" | "center" | "end";
  textRotation: number;
  svgScale: number[];
  svgHorizontalAlignment: "start" | "center" | "end";
  svgVerticalAlignment: "start" | "center" | "end";
  manualOverride?: boolean;
  settingsMode?: "auto" | "manual";
  customHeightMM?: number;
  heightMode?: "auto" | "manual";
}

const FILE_PRINTER = { id: "file", name: "Print to File" };
const DEFAULT_LABEL_SIZE = "62x29";

import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import QRLabelPage from "./pages/QRLabelPage";

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<MainApp />} />
        <Route path="/qr" element={<QRLabelPage />} />
      </Routes>
    </Router>
  );
}

function MainApp() {
  const savedSettings = loadSettings();
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Aina oletusarvot, ei nullia
  const [selectedPrinter, setSelectedPrinter] = useState(
    savedSettings?.selectedPrinter || FILE_PRINTER,
  );
  const [selectedLabelSize, setSelectedLabelSize] = useState<string>(
    savedSettings?.selectedLabelSize || DEFAULT_LABEL_SIZE,
  );
  const [manualOverride] = useState<boolean>(
    savedSettings?.manualOverride || false,
  );
  const [settingsMode, setSettingsMode] = useState<"auto" | "manual">(
    savedSettings?.settingsMode || "auto",
  );
  const [labelWidth, setLabelWidth] = useState(savedSettings?.labelWidth || 62);
  const [labelHeight, setLabelHeight] = useState(
    savedSettings?.labelHeight || 29,
  );
  const [dotsTotalWidth, setDotsTotalWidth] = useState(
    savedSettings?.dotsTotalWidth || 732,
  );
  const [dotsTotalHeight, setDotsTotalHeight] = useState(
    savedSettings?.dotsTotalHeight || 341,
  );
  const [printableLabelWidth, setPrintableLabelWidth] = useState(
    savedSettings?.printableLabelWidth || 696,
  );
  const [printableLabelHeight, setPrintableLabelHeight] = useState(
    savedSettings?.printableLabelHeight || 271,
  );
  const [selectedOrientation, setSelectedOrientation] = useState<string>(
    savedSettings?.selectedOrientation || "standard",
  );
  const [selectedFont, setSelectedFont] = useState<string>(
    savedSettings?.selectedFont || "",
  );
  const [fontSize, setFontSize] = useState<number[]>(
    savedSettings?.fontSize || [80],
  );
  const [labelText, setLabelText] = useState<string>(
    savedSettings?.labelText || "Hello, World!",
  );
  const [qrData, setQrData] = useState<string>(savedSettings?.qrData || "");
  const [svgData, setSvgData] = useState<string | null>(null);
  const [printMode, setPrintMode] = useState<PrintMode>(
    savedSettings?.printMode || "text",
  );
  const [horizontalAlignment, setHorizontalAlignment] = useState<
    "start" | "center" | "end"
  >(savedSettings?.horizontalAlignment || "center");
  const [verticalAlignment, setVerticalAlignment] = useState<
    "start" | "center" | "end"
  >(savedSettings?.verticalAlignment || "center");
  const [textRotation, setTextRotation] = useState<number>(
    savedSettings?.textRotation || 0,
  );
  const [svgScale, setSvgScale] = useState<number[]>(
    savedSettings?.svgScale || [100],
  );
  const [svgHorizontalAlignment, setSvgHorizontalAlignment] = useState<
    "start" | "center" | "end"
  >(savedSettings?.svgHorizontalAlignment || "center");
  const [svgVerticalAlignment, setSvgVerticalAlignment] = useState<
    "start" | "center" | "end"
  >(savedSettings?.svgVerticalAlignment || "center");
  const [customHeightMM, setCustomHeightMM] = useState<number>(
    savedSettings?.customHeightMM || 0,
  );
  const [heightMode, setHeightMode] = useState<"auto" | "manual">(
    savedSettings?.heightMode || "auto",
  );

  // Endless tape detection: both labelHeight (mm) and printableLabelHeight (dots)
  // are 0 for continuous/endless tape. Check both to handle stale defaults.
  const isEndlessTape = labelHeight === 0 || printableLabelHeight === 0;

  // Backend preview integration
  const { previewUrl, isLoading: previewLoading, error: previewError } = usePreview({
    text: labelText,
    labelSize: selectedLabelSize,
    fontFamily: selectedFont,
    fontSize: fontSize[0],
    orientation: selectedOrientation,
    horizontalAlignment,
    verticalAlignment,
    textRotation,
    svgData: printMode === 'svg' ? svgData : null,
    svgScale: svgScale[0] / 100,
    svgHorizontalAlignment,
    svgVerticalAlignment,
    customHeightMM: heightMode === "manual" ? customHeightMM : 0,
    enabled: printMode === 'text' || printMode === 'svg', // Only for text/SVG modes
  });

  useEffect(() => {
    const settings: AppSettings = {
      selectedPrinter,
      selectedLabelSize,
      labelWidth,
      labelHeight,
      dotsTotalWidth,
      dotsTotalHeight,
      printableLabelWidth,
      printableLabelHeight,
      selectedOrientation,
      selectedFont,
      fontSize,
      labelText,
      qrData,
      printMode,
      horizontalAlignment,
      verticalAlignment,
      textRotation,
      svgScale,
      svgHorizontalAlignment,
      svgVerticalAlignment,
      manualOverride,
      settingsMode,
      customHeightMM,
      heightMode,
    };
    saveSettings(settings);
  }, [
    selectedPrinter,
    selectedLabelSize,
    labelWidth,
    labelHeight,
    dotsTotalWidth,
    dotsTotalHeight,
    printableLabelWidth,
    printableLabelHeight,
    selectedOrientation,
    selectedFont,
    fontSize,
    labelText,
    qrData,
    printMode,
    horizontalAlignment,
    verticalAlignment,
    textRotation,
    svgScale,
    svgHorizontalAlignment,
    svgVerticalAlignment,
    customHeightMM,
    heightMode,
  ]);

  const handleLabelSizeChange = useCallback((size: { id: string }) => {
    fetch(`/api/label-sizes/${size.id}`)
      .then((response) => response.json())
      .then((data) => {
        setSelectedLabelSize(data.id);
        setLabelWidth(data.tape_size_width);
        setLabelHeight(data.tape_size_height);
        setDotsTotalWidth(data.dots_total_width);
        setDotsTotalHeight(data.dots_total_height);
        setPrintableLabelWidth(data.dots_printable_width);
        setPrintableLabelHeight(data.dots_printable_height);
      });
  }, []);

  const handleResetSettings = () => {
    if (confirm("Reset all settings to defaults?")) {
      localStorage.removeItem("labelSettings");
      setSelectedPrinter(FILE_PRINTER);
      setSelectedLabelSize(DEFAULT_LABEL_SIZE);
      setLabelWidth(62);
      setLabelHeight(29);
      setDotsTotalWidth(732);
      setDotsTotalHeight(341);
      setPrintableLabelWidth(696);
      setPrintableLabelHeight(271);
      setSelectedOrientation("standard");
      setSelectedFont("");
      setFontSize([80]);
      setLabelText("Hello, World!");
      setQrData("");
      setSvgData(null);
      setPrintMode("text");
      setHorizontalAlignment("center");
      setVerticalAlignment("center");
      setTextRotation(0);
      setSvgScale([100]);
      setSvgHorizontalAlignment("center");
      setSvgVerticalAlignment("center");
      setCustomHeightMM(0);
      setHeightMode("auto");
      setPngFile(null);
    }
  };

  const [pngFile, setPngFile] = useState<File | null>(null);
  const pngInputRef = useRef<HTMLInputElement>(null);

  // Printer recovery state
  const [printerError, setPrinterError] = useState<string | null>(null);
  const [showPrinterDisconnected, setShowPrinterDisconnected] = useState(false);
  
  const {
    isRecovering,
    startBackgroundRecovery,
    stopRecovery,
    manualRetryNow
  } = usePrinterRecovery();

  const handlePrinterRecovered = useCallback((printer: PrinterInfo) => {
    setSelectedPrinter(printer);
    setPrinterError(null);
    setShowPrinterDisconnected(false);
  }, []);

  const handlePrinterRecoveryFailed = useCallback(() => {
    // Recovery attempts exhausted, keep showing the notification for user action
  }, []);

  const handlePrinterRetry = useCallback(() => {
    manualRetryNow(
      (error: string) => {
        setPrinterError(error);
      },
      handlePrinterRecovered
    );
  }, [manualRetryNow, handlePrinterRecovered]);

  const handlePrinterCancel = useCallback(() => {
    stopRecovery();
    setShowPrinterDisconnected(false);
    setPrinterError(null);
  }, [stopRecovery]);

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (file.type === "image/svg+xml") {
      const reader = new FileReader();
      reader.onload = (e) => {
        const text = e.target?.result as string;
        setSvgData(text);
        setPrintMode("svg");
      };
      reader.readAsText(file);
    } else if (file.type === "image/png") {
      setPngFile(file);
      setPrintMode("png");
    }
  };

  const handleClearPng = () => {
    setPngFile(null);
    if (pngInputRef.current) pngInputRef.current.value = "";
    setPrintMode("text");
  };

  const handleClearSvg = () => {
    setSvgData(null);
    if (fileInputRef.current) fileInputRef.current.value = "";
    setPrintMode("text");
  };

  const handlePrint = async () => {
    // Varmista että kaikki pakolliset kentät löytyy
    if (!selectedPrinter?.id || !selectedLabelSize) {
      alert("Please select a printer and label size.");
      return;
    }

    let endpoint = "";
    const customHeight = heightMode === "manual" ? customHeightMM : 0;
    let payload: any = {
      printer: selectedPrinter.id,
      model: selectedPrinter.name,
      label_size: selectedLabelSize,
      ...(customHeight > 0 && { custom_height_mm: customHeight }),
    };

    if (printMode === "text") {
      if (!labelText || !selectedFont) {
        alert("Please enter text and select a font.");
        return;
      }
      endpoint = "/api/print";
      payload = {
        ...payload,
        text: labelText,
        font_family: selectedFont,
        font_size: fontSize[0],
        orientation: selectedOrientation,
        horizontal_alignment: horizontalAlignment,
        vertical_alignment: verticalAlignment,
        text_rotation: textRotation,
      };
    } else if (printMode === "qr") {
      if (!qrData) {
        alert("Please enter data for the QR code.");
        return;
      }
      endpoint = "/api/print_qr";
      payload = { ...payload, data: qrData };
    } else if (printMode === "svg") {
      if (!svgData) {
        alert("Please load an SVG file.");
        return;
      }
      endpoint = "/api/print_svg";
      payload = {
        ...payload,
        svg_data: svgData,
        orientation: selectedOrientation,
        scale: svgScale[0] / 100,
      };
    } else if (printMode === "png") {
      if (!pngFile) {
        alert("Please select a PNG file.");
        return;
      }
      endpoint = "/api/print_png";
      try {
        const base64Data = await new Promise<string>((resolve, reject) => {
          const reader = new FileReader();
          reader.readAsDataURL(pngFile);
          reader.onload = () =>
            resolve((reader.result as string).split(",")[1]);
          reader.onerror = (error) => reject(error);
        });
        payload = {
          ...payload,
          png_data: base64Data,
        };
      } catch (error) {
        console.error("Error converting file to base64:", error);
        alert("Could not process PNG file. See console for details.");
        return;
      }
    }

    try {
      console.log("Sending print request with payload:", payload);
      const response = await fetch(endpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });
      if (response.ok) {
        alert("Print job sent successfully!");
      } else {
        const errorData = await response.json();
        const errorMessage = errorData.error || JSON.stringify(errorData);
        
        // Check if it's a USB device error from API response
        if (errorMessage.includes('USB device not found') || errorMessage.includes('device not found')) {
          setPrinterError(errorMessage);
          setShowPrinterDisconnected(true);
          
          // Start background recovery process
          startBackgroundRecovery(
            (error: string) => {
              setPrinterError(error);
            },
            handlePrinterRecovered,
            handlePrinterRecoveryFailed
          );
        } else {
          alert(`Error: ${errorMessage}`);
        }
      }
    } catch (error) {
      console.error("Failed to send print job:", error);
      
      // Check if it's a USB device error
      const errorMessage = error instanceof Error ? error.message : JSON.stringify(error);
      if (errorMessage.includes('USB device not found') || errorMessage.includes('device not found')) {
        setPrinterError(errorMessage);
        setShowPrinterDisconnected(true);
        
        // Start background recovery process
        startBackgroundRecovery(
          (error: string) => {
            setPrinterError(error);
          },
          handlePrinterRecovered,
          handlePrinterRecoveryFailed
        );
      } else {
        alert("Failed to send print job. See console for details.");
      }
    }
  };

  const renderContentSpecificControls = () => {
    switch (printMode) {
      case "text":
        return (
          <>
            <div>
              <div className="flex justify-between items-center mb-2">
                <TextAlignmentSelector
                  onHorizontalChange={setHorizontalAlignment}
                  onVerticalChange={setVerticalAlignment}
                  horizontalValue={horizontalAlignment}
                  verticalValue={verticalAlignment}
                  onTextRotationChange={setTextRotation}
                  textRotationValue={textRotation}
                  onOrientationChange={
                    setSelectedOrientation as (
                      value: "standard" | "rotated",
                    ) => void
                  }
                  orientationValue={
                    selectedOrientation as "standard" | "rotated"
                  }
                />
              </div>
              <Textarea
                id="label-text"
                value={labelText}
                onChange={(e) => setLabelText(e.target.value)}
                placeholder="Enter text for the label"
                rows={4}
              />
            </div>
            <div>
              <Label htmlFor="font-select">Font Family</Label>
              <FontSelector onSelectFont={setSelectedFont} />
            </div>
            <div>
              <Label htmlFor="font-size">Font Size: {fontSize[0]}</Label>
              <Slider
                id="font-size"
                min={10}
                max={72}
                step={1}
                value={fontSize}
                onValueChange={setFontSize}
              />
            </div>
          </>
        );
      case "qr":
        return <QRCodeGenerator qrData={qrData} onQrDataChange={setQrData} />;
      case "svg":
        return (
          <div className="space-y-4">
            <input
              type="file"
              accept=".svg"
              ref={fileInputRef}
              onChange={handleFileChange}
              style={{ display: "none" }}
            />
            <Button
              variant="outline"
              onClick={() => fileInputRef.current?.click()}
              className="w-full"
            >
              Load SVG
            </Button>
            {svgData && (
              <>
                <Button
                  variant="destructive"
                  onClick={handleClearSvg}
                  className="w-full"
                >
                  Clear SVG
                </Button>
                <div>
                  <Label htmlFor="svg-scale">SVG Scale: {svgScale[0]}%</Label>
                  <Slider
                    id="svg-scale"
                    min={10}
                    max={200}
                    step={1}
                    value={svgScale}
                    onValueChange={setSvgScale}
                  />
                </div>
                <div>
                  <TextAlignmentSelector
                    onHorizontalChange={setSvgHorizontalAlignment}
                    onVerticalChange={setSvgVerticalAlignment}
                    horizontalValue={svgHorizontalAlignment}
                    verticalValue={svgVerticalAlignment}
                    onTextRotationChange={() => {}}
                    textRotationValue={0}
                    onOrientationChange={() => {}}
                    orientationValue={"standard"}
                  />
                </div>
              </>
            )}
          </div>
        );
      case "png":
        return (
          <div className="space-y-4">
            <input
              type="file"
              accept="image/png"
              ref={pngInputRef}
              onChange={handleFileChange}
              style={{ display: "none" }}
            />
            <Button
              variant="outline"
              onClick={() => pngInputRef.current?.click()}
              className="w-full"
            >
              Load PNG
            </Button>
            {pngFile && (
              <>
                <Button
                  variant="destructive"
                  onClick={handleClearPng}
                  className="w-full"
                >
                  Clear PNG
                </Button>
                <div className="flex justify-center">
                  <img
                    src={URL.createObjectURL(pngFile)}
                    alt="Preview"
                    className="max-h-40"
                  />
                </div>
              </>
            )}
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <div className="min-h-screen">
      <header className="border-b">
        <div className="container flex h-16 items-center px-4">
          <h1 className="text-2xl font-bold">Brother QL Label Designer</h1>
        </div>
      </header>
      <main className="container flex-1 px-4 py-6">
        {/* Printer Disconnection Notification */}
        {showPrinterDisconnected && printerError && (
          <PrinterDisconnectedNotification
            error={printerError}
            onRetry={handlePrinterRetry}
            onCancel={handlePrinterCancel}
            isRetrying={isRecovering}
          />
        )}
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <div className="lg:col-span-1 space-y-8">
            <Card>
              <CardHeader>
                <CardTitle>Label Content</CardTitle>
                <CardDescription>
                  Select a content mode and customize your label.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <RadioGroup
                  value={printMode}
                  onValueChange={(value: string) =>
                    setPrintMode(value as PrintMode)
                  }
                  className="flex space-x-4"
                >
                  <div className="flex items-center space-x-2">
                    <RadioGroupItem value="text" id="mode-text" />
                    <Label htmlFor="mode-text">Text</Label>
                  </div>
                  <div className="flex items-center space-x-2">
                    <RadioGroupItem value="qr" id="mode-qr" />
                    <Label htmlFor="mode-qr">QR Code</Label>
                  </div>
                  <div className="flex items-center space-x-2">
                    <RadioGroupItem value="svg" id="mode-svg" />
                    <Label htmlFor="mode-svg">SVG</Label>
                  </div>
                  <div className="flex items-center space-x-2">
                    <RadioGroupItem value="png" id="mode-png" />
                    <Label htmlFor="mode-png">PNG</Label>
                  </div>
                </RadioGroup>
                <div className="pt-4">{renderContentSpecificControls()}</div>
                {isEndlessTape && (
                  <div className="border border-gray-200 rounded-lg p-3 mt-4 bg-gray-50 dark:bg-gray-900 dark:border-gray-700">
                    <Label className="text-sm font-medium mb-2 block">Tape Length</Label>
                    <RadioGroup
                      value={heightMode}
                      onValueChange={(value: string) =>
                        setHeightMode(value as "auto" | "manual")
                      }
                      className="flex space-x-4 mb-2"
                    >
                      <div className="flex items-center space-x-2">
                        <RadioGroupItem value="auto" id="height-auto" />
                        <Label htmlFor="height-auto" className="text-sm">Auto (fit to content)</Label>
                      </div>
                      <div className="flex items-center space-x-2">
                        <RadioGroupItem value="manual" id="height-manual" />
                        <Label htmlFor="height-manual" className="text-sm">Manual</Label>
                      </div>
                    </RadioGroup>
                    {heightMode === "manual" && (
                      <div className="flex items-center space-x-2 mt-2">
                        <Label htmlFor="custom-height" className="text-sm whitespace-nowrap">Height (mm):</Label>
                        <input
                          id="custom-height"
                          type="number"
                          min={10}
                          max={2000}
                          value={customHeightMM || ""}
                          onChange={(e) => setCustomHeightMM(Math.max(0, Math.min(2000, Number(e.target.value))))}
                          placeholder="e.g. 100"
                          className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                        />
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
              <CardFooter className="space-y-2 flex-col">
                <Button onClick={handlePrint} className="w-full">
                  Print Label
                </Button>
                <Button
                  variant="outline"
                  onClick={handleResetSettings}
                  className="w-full"
                >
                  Reset Settings
                </Button>
              </CardFooter>
            </Card>
            <Accordion type="single" collapsible className="w-full">
              <AccordionItem value="item-1">
                <AccordionTrigger>
                  <h3 className="text-lg font-medium">Printer Settings</h3>
                </AccordionTrigger>
                <AccordionContent>
                  <div className="grid gap-4 pt-4">
                    <ModeToggle
                      value={settingsMode}
                      onValueChange={setSettingsMode}
                    />

                    <div>
                      <PrinterSelector
                        value={selectedPrinter?.id}
                        onSelectPrinter={setSelectedPrinter}
                        onDetectLabelSize={(labelSizeId) => {
                          if (settingsMode === "auto") {
                            console.log("Auto-detected label size:", labelSizeId);
                            handleLabelSizeChange({ id: labelSizeId });
                          }
                        }}
                        manualOverride={settingsMode === "manual"}
                      />
                    </div>

                    {/* Manual Mode Section - Show only manual controls when in manual mode */}
                    {settingsMode === "manual" && (
                      <div className="border border-gray-200 rounded-lg p-4 bg-gray-50">
                        <h4 className="text-sm font-medium mb-2 text-gray-800">
                          Manual Settings
                        </h4>
                        <p className="text-xs text-gray-600 mb-3">
                          Manually select label size and orientation.
                        </p>
                        <div className="space-y-4">
                          <div>
                            <Label htmlFor="label-size-select">
                              Select Label Size
                            </Label>
                            <LabelSizeSelector
                              value={selectedLabelSize}
                              onLabelSizeChange={handleLabelSizeChange}
                            />
                          </div>
                          <div>
                            <Label htmlFor="orientation-select">
                              Orientation
                            </Label>
                            <Select
                              value={selectedOrientation}
                              onValueChange={setSelectedOrientation}
                              disabled={printMode === "qr"}
                            >
                              <SelectTrigger
                                id="orientation-select"
                                className="w-full"
                              >
                                <SelectValue placeholder="Select orientation" />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="standard">
                                  Standard
                                </SelectItem>
                                <SelectItem value="rotated">Rotated</SelectItem>
                              </SelectContent>
                            </Select>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </div>
          <div className="lg:col-span-2">
            <Card className="h-full">
              <CardHeader>
                <CardTitle>Label Preview</CardTitle>
                {previewLoading && (
                  <CardDescription className="text-blue-600">
                    Loading preview...
                  </CardDescription>
                )}
                {previewError && (
                  <CardDescription className="text-red-600">
                    Preview error: {previewError}
                  </CardDescription>
                )}
              </CardHeader>
              <CardContent className="flex items-center justify-center h-full min-h-[500px]">
                <LabelPreview
                  labelText={printMode === "text" ? labelText : ""}
                  svgData={printMode === "svg" ? svgData : null}
                  qrData={printMode === "qr" ? qrData : ""}
                  pngData={printMode === "png" ? pngFile : null}
                  selectedFont={selectedFont}
                  fontSize={fontSize[0]}
                  labelWidth={labelWidth}
                  labelHeight={labelHeight}
                  dotsTotalWidth={dotsTotalWidth}
                  dotsTotalHeight={dotsTotalHeight}
                  printableLabelWidth={printableLabelWidth}
                  printableLabelHeight={printableLabelHeight}
                  orientation={selectedOrientation as "standard" | "rotated"}
                  horizontalAlignment={horizontalAlignment}
                  verticalAlignment={verticalAlignment}
                  textRotation={textRotation}
                  svgScale={svgScale[0] / 100}
                  svgHorizontalAlignment={svgHorizontalAlignment}
                  svgVerticalAlignment={svgVerticalAlignment}
                  previewUrl={(printMode === 'text' || printMode === 'svg') ? previewUrl : null}
                  customHeightMM={customHeightMM}
                  heightMode={heightMode}
                />
              </CardContent>
            </Card>
          </div>
        </div>
      </main>
    </div>
  );
}

export default App;
