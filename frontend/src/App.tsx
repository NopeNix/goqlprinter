import { useState, useCallback, useRef } from "react";
import {
  Card,
  CardContent,
  CardFooter,
} from "./components/ui/card";
import { Button } from "./components/ui/button";
import { Textarea } from "./components/ui/textarea";
import { Label } from "./components/ui/label";
import { Slider } from "./components/ui/slider";
import PrinterDisconnectedNotification from "./components/PrinterDisconnectedNotification";
import PrinterStatusBar from "./components/PrinterStatusBar";
import AdvancedSettingsPanel from "./components/AdvancedSettingsPanel";
import FontSelector from "./components/FontSelector";
import LabelPreview from "./components/LabelPreview";
import QRCodeGenerator from "./components/QRCodeGenerator";
import TextAlignmentSelector from "./components/TextAlignmentSelector";
import { usePrinterStatus, type PrinterInfo } from "./hooks/usePrinterStatus";
import { usePreview } from "./hooks/usePreview";
import { useLabelSettings } from "./hooks/useLabelSettings";
import type { PrintMode } from "./hooks/useLabelSettings";
import { usePrintJob } from "./hooks/usePrintJob";
import { DOTS_PER_MM } from "./constants";
import { ToggleGroup, ToggleGroupItem } from "./components/ui/toggle-group";
import { Toaster } from "./components/ui/sonner";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "./components/ui/alert-dialog";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "./components/ui/tabs";

import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import QRLabelPage from "./pages/QRLabelPage";
import { labelApi } from "./api/endpoints";
import ErrorBoundary from "./components/ErrorBoundary";

function App() {
  return (
    <ErrorBoundary>
      <Router>
        <Toaster richColors position="top-right" />
        <Routes>
          <Route path="/" element={<MainApp />} />
          <Route path="/qr" element={<QRLabelPage />} />
        </Routes>
      </Router>
    </ErrorBoundary>
  );
}

function MainApp() {
  const { settings, dispatch, isEndlessTape } = useLabelSettings();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleLabelSizeChange = useCallback((size: { id: string }) => {
    labelApi.size(size.id).then((data) => {
      dispatch({
        type: "SET_LABEL_SIZE",
        payload: {
          id: data.id,
          dimensions: {
            labelWidth: data.tape_size_width,
            labelHeight: data.tape_size_height,
            dotsTotalWidth: data.dots_total_width,
            dotsTotalHeight: data.dots_total_height,
            printableLabelWidth: data.dots_printable_width,
            printableLabelHeight: data.dots_printable_height,
          },
        },
      });
    });
  }, [dispatch]);

  const [svgData, setSvgData] = useState<string | null>(null);
  const [pngFile, setPngFile] = useState<File | null>(null);
  const pngInputRef = useRef<HTMLInputElement>(null);

  // Backend preview integration
  const { previewUrl, isLoading: previewLoading, error: previewError } = usePreview({
    text: settings.labelText,
    labelSize: settings.selectedLabelSize,
    fontFamily: settings.selectedFont,
    fontSize: settings.fontSize[0],
    orientation: settings.selectedOrientation,
    horizontalAlignment: settings.horizontalAlignment,
    verticalAlignment: settings.verticalAlignment,
    textRotation: settings.textRotation,
    svgData: settings.printMode === 'svg' ? svgData : null,
    svgScale: settings.svgScale[0] / 100,
    svgHorizontalAlignment: settings.svgHorizontalAlignment,
    svgVerticalAlignment: settings.svgVerticalAlignment,
    customHeightMM: settings.heightMode === "manual" ? settings.customHeightMM : 0,
    enabled: settings.printMode === 'text' || settings.printMode === 'svg',
  });

  const [showAdvancedSettings, setShowAdvancedSettings] = useState(false);
  const [showResetDialog, setShowResetDialog] = useState(false);

  const setSelectedPrinter = useCallback(
    (printer: PrinterInfo) => {
      dispatch({ type: "SET_PRINTER", payload: printer });
    },
    [dispatch],
  );

  const printerState = usePrinterStatus(settings.selectedPrinter, setSelectedPrinter, {
    settingsMode: settings.settingsMode,
    onLabelSizeChange: handleLabelSizeChange,
  });

  const handleResetSettings = () => {
    setShowResetDialog(true);
  };

  const confirmResetSettings = () => {
    localStorage.removeItem("labelSettings");
    dispatch({ type: "RESET" });
    setSvgData(null);
    setPngFile(null);
    setShowResetDialog(false);
  };

  const printJob = usePrintJob({
    settings,
    svgData,
    pngFile,
    onPrinterRecovered: (printer) => dispatch({ type: "SET_PRINTER", payload: printer }),
  });

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (file.type === "image/svg+xml") {
      const reader = new FileReader();
      reader.onload = (e) => {
        const text = e.target?.result as string;
        setSvgData(text);
        dispatch({ type: "SET_PRINT_MODE", payload: "svg" });
      };
      reader.readAsText(file);
    } else if (file.type === "image/png") {
      setPngFile(file);
      dispatch({ type: "SET_PRINT_MODE", payload: "png" });
    }
  };

  const handleClearPng = () => {
    setPngFile(null);
    if (pngInputRef.current) pngInputRef.current.value = "";
    dispatch({ type: "SET_PRINT_MODE", payload: "text" });
  };

  const handleClearSvg = () => {
    setSvgData(null);
    if (fileInputRef.current) fileInputRef.current.value = "";
    dispatch({ type: "SET_PRINT_MODE", payload: "text" });
  };

  const handleSelectFont = useCallback(
    (font: string) => dispatch({ type: "SET_FONT", payload: font }),
    [dispatch],
  );

  const renderContentSpecificControls = () => {
    switch (settings.printMode) {
      case "text":
        return (
          <>
            <Textarea
              id="label-text"
              value={settings.labelText}
              onChange={(e) => dispatch({ type: "SET_TEXT", payload: e.target.value })}
              placeholder="Label text"
              rows={3}
            />
            <FontSelector onSelectFont={handleSelectFont} />
            <div className="flex items-center gap-2">
              <Slider
                id="font-size"
                min={10}
                max={72}
                step={1}
                value={settings.fontSize}
                onValueChange={(v) => dispatch({ type: "SET_FONT_SIZE", payload: v })}
                className="flex-1"
              />
              <input
                type="number"
                min={10}
                max={72}
                value={settings.fontSize[0]}
                onChange={(e) => dispatch({ type: "SET_FONT_SIZE", payload: [Math.max(10, Math.min(72, Number(e.target.value)))] })}
                className="h-7 w-12 rounded-md border border-input bg-transparent px-1 text-xs text-center tabular-nums shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              />
            </div>
          </>
        );
      case "qr":
        return <QRCodeGenerator qrData={settings.qrData} onQrDataChange={(v: string) => dispatch({ type: "SET_QR_DATA", payload: v })} />;
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
                  <Label htmlFor="svg-scale">SVG Scale: {settings.svgScale[0]}%</Label>
                  <Slider
                    id="svg-scale"
                    min={10}
                    max={200}
                    step={1}
                    value={settings.svgScale}
                    onValueChange={(v) => dispatch({ type: "SET_SVG_SCALE", payload: v })}
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

  const contentColumn = (
    <Card>
      <CardContent className="space-y-3 pt-4">
        <ToggleGroup
          type="single"
          value={settings.printMode}
          onValueChange={(value: string) => {
            if (value) dispatch({ type: "SET_PRINT_MODE", payload: value as PrintMode });
          }}
          variant="outline"
          size="sm"
        >
          <ToggleGroupItem value="text" aria-label="Text mode">Text</ToggleGroupItem>
          <ToggleGroupItem value="qr" aria-label="QR mode">QR</ToggleGroupItem>
          <ToggleGroupItem value="svg" aria-label="SVG mode">SVG</ToggleGroupItem>
          <ToggleGroupItem value="png" aria-label="PNG mode">PNG</ToggleGroupItem>
        </ToggleGroup>
        {renderContentSpecificControls()}
      </CardContent>
      <CardFooter className="hidden md:block pt-0">
        <Button onClick={printJob.handlePrint} className="w-full">
          Print Label
        </Button>
      </CardFooter>
    </Card>
  );

  const isLabelRotated = settings.selectedOrientation === "rotated";
  const printableWidthMm = (settings.dimensions.printableLabelWidth / DOTS_PER_MM).toFixed(1);
  const displayHeightMm = (settings.heightMode === "manual" && settings.customHeightMM > 0)
    ? settings.customHeightMM.toFixed(1)
    : (settings.dimensions.printableLabelHeight / DOTS_PER_MM).toFixed(1);
  const printableAreaLabel = isLabelRotated
    ? `${displayHeightMm} × ${printableWidthMm} mm`
    : `${printableWidthMm} × ${displayHeightMm} mm`;

  const alignmentControls = () => {
    if (settings.printMode === "text") {
      return (
        <TextAlignmentSelector
          onHorizontalChange={(v: "start" | "center" | "end") => dispatch({ type: "SET_HORIZONTAL_ALIGNMENT", payload: v })}
          onVerticalChange={(v: "start" | "center" | "end") => dispatch({ type: "SET_VERTICAL_ALIGNMENT", payload: v })}
          horizontalValue={settings.horizontalAlignment}
          verticalValue={settings.verticalAlignment}
          onTextRotationChange={(v: number) => dispatch({ type: "SET_TEXT_ROTATION", payload: v })}
          textRotationValue={settings.textRotation}
          onOrientationChange={
            ((value: "standard" | "rotated") => dispatch({ type: "SET_ORIENTATION", payload: value })) as (value: "standard" | "rotated") => void
          }
          orientationValue={settings.selectedOrientation as "standard" | "rotated"}
        />
      );
    }
    if (settings.printMode === "svg" && svgData) {
      return (
        <TextAlignmentSelector
          onHorizontalChange={(v: "start" | "center" | "end") => dispatch({ type: "SET_SVG_HORIZONTAL_ALIGNMENT", payload: v })}
          onVerticalChange={(v: "start" | "center" | "end") => dispatch({ type: "SET_SVG_VERTICAL_ALIGNMENT", payload: v })}
          horizontalValue={settings.svgHorizontalAlignment}
          verticalValue={settings.svgVerticalAlignment}
          onTextRotationChange={() => {}}
          textRotationValue={0}
          onOrientationChange={() => {}}
          orientationValue={"standard"}
        />
      );
    }
    return null;
  };

  const previewColumn = (
    <Card className="flex flex-col">
      <div className="flex items-center gap-2 px-4 pt-3 pb-0">
        {isEndlessTape ? (
          <span className="text-xs text-muted-foreground flex items-center gap-0.5">
            {isLabelRotated ? (
              <>
                {settings.heightMode === "manual" ? (
                  <input
                    type="number"
                    min={10}
                    max={2000}
                    value={settings.customHeightMM || ""}
                    onChange={(e) => dispatch({ type: "SET_CUSTOM_HEIGHT_MM", payload: Math.max(0, Math.min(2000, Number(e.target.value))) })}
                    placeholder="mm"
                    className="h-5 w-14 rounded border border-input bg-transparent px-1 text-xs text-center tabular-nums focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  />
                ) : (
                  <span>{displayHeightMm}</span>
                )}
                <span> × {printableWidthMm} mm</span>
              </>
            ) : (
              <>
                <span>{printableWidthMm} × </span>
                {settings.heightMode === "manual" ? (
                  <input
                    type="number"
                    min={10}
                    max={2000}
                    value={settings.customHeightMM || ""}
                    onChange={(e) => dispatch({ type: "SET_CUSTOM_HEIGHT_MM", payload: Math.max(0, Math.min(2000, Number(e.target.value))) })}
                    placeholder="mm"
                    className="h-5 w-14 rounded border border-input bg-transparent px-1 text-xs text-center tabular-nums focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  />
                ) : (
                  <span>{displayHeightMm}</span>
                )}
                <span> mm</span>
              </>
            )}
            <button
              type="button"
              onClick={() => dispatch({ type: "SET_HEIGHT_MODE", payload: settings.heightMode === "auto" ? "manual" : "auto" })}
              className={`ml-1 text-[10px] px-1.5 py-0.5 rounded-full border transition-colors cursor-pointer ${
                settings.heightMode === "manual"
                  ? "bg-foreground text-background border-foreground hover:bg-foreground/80"
                  : "bg-muted border-border hover:bg-accent hover:border-accent-foreground/20"
              }`}
              title={settings.heightMode === "auto" ? "Switch to fixed height" : "Switch to auto height"}
            >
              {settings.heightMode === "auto" ? "auto" : "fixed"}
            </button>
          </span>
        ) : (
          <span className="text-xs text-muted-foreground">{printableAreaLabel}</span>
        )}
        {previewLoading && (
          <span className="text-xs text-blue-600 ml-auto">Loading...</span>
        )}
        {previewError && (
          <span className="text-xs text-red-600 ml-auto truncate max-w-[200px]" title={previewError}>
            {previewError}
          </span>
        )}
      </div>
      {alignmentControls() && (
        <div className="flex justify-center px-4 pt-2">
          {alignmentControls()}
        </div>
      )}
      <CardContent className="flex flex-1 items-center justify-center overflow-auto max-w-full pt-2 pb-4">
        <LabelPreview
          labelText={settings.printMode === "text" ? settings.labelText : ""}
          svgData={settings.printMode === "svg" ? svgData : null}
          qrData={settings.printMode === "qr" ? settings.qrData : ""}
          pngData={settings.printMode === "png" ? pngFile : null}
          selectedFont={settings.selectedFont}
          fontSize={settings.fontSize[0]}
          labelWidth={settings.dimensions.labelWidth}
          labelHeight={settings.dimensions.labelHeight}
          dotsTotalWidth={settings.dimensions.dotsTotalWidth}
          dotsTotalHeight={settings.dimensions.dotsTotalHeight}
          printableLabelWidth={settings.dimensions.printableLabelWidth}
          printableLabelHeight={settings.dimensions.printableLabelHeight}
          orientation={settings.selectedOrientation as "standard" | "rotated"}
          horizontalAlignment={settings.horizontalAlignment}
          verticalAlignment={settings.verticalAlignment}
          textRotation={settings.textRotation}
          svgScale={settings.svgScale[0] / 100}
          svgHorizontalAlignment={settings.svgHorizontalAlignment}
          svgVerticalAlignment={settings.svgVerticalAlignment}
          previewUrl={(settings.printMode === 'text' || settings.printMode === 'svg') ? previewUrl : null}
          customHeightMM={settings.customHeightMM}
          heightMode={settings.heightMode}
        />
      </CardContent>
    </Card>
  );

  return (
    <div className="min-h-screen">
      <PrinterStatusBar
        printerName={settings.selectedPrinter?.name || "No printer"}
        labelSize={settings.selectedLabelSize}
        labelWidth={settings.dimensions.labelWidth}
        labelHeight={settings.dimensions.labelHeight}
        status={printerState.status}
        statusDetail={printerState.statusDetail}
        expanded={showAdvancedSettings}
        onClick={() => setShowAdvancedSettings(!showAdvancedSettings)}
      />
      <main className="container flex-1 px-4 py-6 pb-20 md:pb-0">
        {/* Printer Disconnection Notification */}
        {printJob.showPrinterDisconnected && printJob.printerError && (
          <PrinterDisconnectedNotification
            error={printJob.printerError}
            onRetry={printJob.handlePrinterRetry}
            onCancel={printJob.handlePrinterCancel}
            isRetrying={printJob.isRecovering}
          />
        )}

        {showAdvancedSettings && (
          <AdvancedSettingsPanel
            printers={printerState.printers}
            selectedPrinter={settings.selectedPrinter}
            onSelectPrinter={printerState.selectPrinter}
            settingsMode={settings.settingsMode}
            onSettingsModeChange={(mode: "auto" | "manual") => dispatch({ type: "SET_SETTINGS_MODE", payload: mode })}
            selectedLabelSize={settings.selectedLabelSize}
            onLabelSizeChange={handleLabelSizeChange}
            selectedOrientation={settings.selectedOrientation}
            onOrientationChange={(v: string) => dispatch({ type: "SET_ORIENTATION", payload: v })}
            printMode={settings.printMode}
            onResetSettings={handleResetSettings}
            loading={printerState.loading}
            refreshDebounced={printerState.refreshDebounced}
            onRefresh={printerState.refresh}
          />
        )}

        {/* Mobile: tabbed layout */}
        <div className="md:hidden">
          <Tabs defaultValue="content">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="content">Content</TabsTrigger>
              <TabsTrigger value="preview">Preview</TabsTrigger>
            </TabsList>
            <TabsContent value="content" className="space-y-8">
              {contentColumn}
            </TabsContent>
            <TabsContent value="preview">
              {previewColumn}
            </TabsContent>
          </Tabs>
        </div>

        {/* Desktop: balanced two-column layout, matched height */}
        <div className="hidden md:grid gap-6 md:grid-cols-2">
          {contentColumn}
          {previewColumn}
        </div>
      </main>

      {/* Sticky mobile print button */}
      <div className="md:hidden fixed bottom-0 left-0 right-0 p-4 bg-background border-t z-50">
        <Button onClick={printJob.handlePrint} className="w-full">
          Print Label
        </Button>
      </div>

      <AlertDialog open={showResetDialog} onOpenChange={setShowResetDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Reset Settings</AlertDialogTitle>
            <AlertDialogDescription>
              Reset all settings to defaults? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmResetSettings}>
              Reset
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

export default App;
