import { useState, useEffect, useCallback } from "react";
import { Toaster } from "../components/ui/sonner";
import { Button } from "../components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "../components/ui/accordion";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "../components/ui/alert-dialog";
import QRCodeGenerator from "../components/QRCodeGenerator";
import PrinterDisconnectedNotification from "../components/PrinterDisconnectedNotification";
import LabelSizeSelector from "../components/LabelSizeSelector";
import ModeToggle from "../components/ModeToggle";
import PrinterStatusBar from "../components/PrinterStatusBar";
import { usePrinterStatus } from "../hooks/usePrinterStatus";
import { usePrintJob } from "../hooks/usePrintJob";
import { DEFAULT_SETTINGS } from "../hooks/useLabelSettings";
import type { LabelSettings } from "../hooks/useLabelSettings";
import { RefreshCw } from "lucide-react";

interface Settings {
  selectedPrinter: { id: string; name: string };
  selectedLabelSize: string;
  qrData: string;
  settingsMode?: "auto" | "manual";
}

const DEFAULT_PRINTER = { id: "file", name: "Print to File" };
const DEFAULT_LABEL_SIZE = "62x29";

function loadSettings(): Settings {
  if (typeof window === 'undefined') {
    return {
      selectedPrinter: DEFAULT_PRINTER,
      selectedLabelSize: DEFAULT_LABEL_SIZE,
      qrData: "",
      settingsMode: "auto"
    };
  }

  const saved = localStorage.getItem("qrLabelSettings");
  if (saved) {
    try {
      const parsed = JSON.parse(saved);
      return {
        ...parsed,
        settingsMode: parsed.settingsMode || "auto"
      };
    } catch {
      localStorage.removeItem("qrLabelSettings");
    }
  }
  return {
    selectedPrinter: DEFAULT_PRINTER,
    selectedLabelSize: DEFAULT_LABEL_SIZE,
    qrData: "",
    settingsMode: "auto"
  };
}

function saveSettings(settings: Settings) {
  localStorage.setItem("qrLabelSettings", JSON.stringify(settings));
}

export default function QRLabelPage() {
  const [settings, setSettings] = useState<Settings>(loadSettings());
  const { selectedPrinter, selectedLabelSize, qrData, settingsMode = "auto" } = settings;

  const setSelectedPrinter = useCallback((printer: { id: string; name: string }) => {
    setSettings(prev => ({ ...prev, selectedPrinter: printer }));
  }, []);

  const setSelectedLabelSize = useCallback((size: { id: string }) => {
    setSettings(prev => ({ ...prev, selectedLabelSize: size.id }));
  }, []);

  const printerState = usePrinterStatus(selectedPrinter, setSelectedPrinter, {
    settingsMode,
    onLabelSizeChange: setSelectedLabelSize,
  });

  // Construct a LabelSettings object for usePrintJob
  const labelSettings: LabelSettings = {
    ...DEFAULT_SETTINGS,
    selectedPrinter,
    selectedLabelSize,
    printMode: "qr",
    qrData,
    settingsMode: settingsMode || "auto",
  };

  const printJob = usePrintJob({
    settings: labelSettings,
    svgData: null,
    pngFile: null,
    onPrinterRecovered: (printer) => setSettings(prev => ({ ...prev, selectedPrinter: printer })),
  });

  useEffect(() => {
    saveSettings(settings);
  }, [settings]);

  const setSettingsMode = (mode: "auto" | "manual") => {
    setSettings(prev => ({ ...prev, settingsMode: mode }));
  };

  const setQrData = (data: string) => {
    setSettings(prev => ({ ...prev, qrData: data }));
  };

  const [showResetDialog, setShowResetDialog] = useState(false);

  const handleResetSettings = () => {
    setShowResetDialog(true);
  };

  const confirmResetSettings = () => {
    setSettings({
      selectedPrinter: DEFAULT_PRINTER,
      selectedLabelSize: DEFAULT_LABEL_SIZE,
      qrData: ""
    });
    setShowResetDialog(false);
  };

  return (
    <div className="container mx-auto p-4 max-w-2xl">
      <Toaster richColors position="top-right" />
      {/* Printer Disconnection Notification */}
      {printJob.showPrinterDisconnected && printJob.printerError && (
        <PrinterDisconnectedNotification
          error={printJob.printerError}
          onRetry={printJob.handlePrinterRetry}
          onCancel={printJob.handlePrinterCancel}
          isRetrying={printJob.isRecovering}
        />
      )}

      <h1 className="text-2xl font-bold mb-6">QR Label Printer</h1>

      <div className="space-y-6">
        <QRCodeGenerator
          qrData={qrData}
          onQrDataChange={setQrData}
        />

        <PrinterStatusBar
          printerName={selectedPrinter?.name || "No printer"}
          labelSize={selectedLabelSize}
          labelWidth={0}
          labelHeight={0}
          status={printerState.status}
          statusDetail={printerState.statusDetail}
        />

        <Accordion type="single" collapsible>
          <AccordionItem value="printer-settings">
            <AccordionTrigger className="font-medium">
              Printer Settings
            </AccordionTrigger>
            <AccordionContent className="space-y-4 pt-4">
              <ModeToggle
                value={settingsMode}
                onValueChange={setSettingsMode}
              />

              <div className="flex items-center gap-2">
                <div className="flex-1">
                  <label className="block mb-2 text-sm font-medium">Printer</label>
                  <Select value={selectedPrinter?.id} onValueChange={printerState.selectPrinter}>
                    <SelectTrigger className="h-9 text-sm">
                      <SelectValue placeholder="Select printer" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="file">Print to File</SelectItem>
                      {printerState.printers.map((p) => (
                        <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <button
                  type="button"
                  onClick={printerState.refresh}
                  disabled={printerState.loading || printerState.refreshDebounced}
                  className="mt-6 inline-flex items-center justify-center h-9 w-9 rounded-md border border-input bg-background hover:bg-accent transition-colors disabled:opacity-50"
                  title="Refresh printers"
                >
                  <RefreshCw className={`h-3.5 w-3.5 ${printerState.loading ? "animate-spin" : ""}`} />
                </button>
              </div>

              {/* Manual Mode Section */}
              {settingsMode === "manual" && (
                <div>
                  <label className="block mb-2 text-sm font-medium">Label Size</label>
                  <LabelSizeSelector
                    value={selectedLabelSize}
                    onLabelSizeChange={setSelectedLabelSize}
                  />
                </div>
              )}
            </AccordionContent>
          </AccordionItem>
        </Accordion>

        <div className="space-y-2">
          <Button
            onClick={printJob.handlePrint}
            className="w-full"
          >
            Print QR Label
          </Button>
          <Button
            variant="outline"
            onClick={handleResetSettings}
            className="w-full"
          >
            Reset Settings
          </Button>
        </div>
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
