"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "../components/ui/button";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "../components/ui/accordion";
import QRCodeGenerator from "../components/QRCodeGenerator";
import PrinterSelector from "../components/PrinterSelector";
import PrinterDisconnectedNotification from "../components/PrinterDisconnectedNotification";
import LabelSizeSelector from "../components/LabelSizeSelector";
import ModeToggle from "../components/ModeToggle";
import usePrinterRecovery, { type PrinterInfo } from "../hooks/usePrinterRecovery";

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
    const parsed = JSON.parse(saved);
    return {
      ...parsed,
      settingsMode: parsed.settingsMode || "auto"
    };
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
    setSettings(prev => ({ ...prev, selectedPrinter: printer }));
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

  useEffect(() => {
    saveSettings(settings);
  }, [settings]);

  const setSelectedPrinter = (printer: { id: string; name: string }) => {
    setSettings(prev => ({ ...prev, selectedPrinter: printer }));
  };

  const setSelectedLabelSize = (size: { id: string }) => {
    setSettings(prev => ({ ...prev, selectedLabelSize: size.id }));
  };

  const setSettingsMode = (mode: "auto" | "manual") => {
    setSettings(prev => ({ ...prev, settingsMode: mode }));
  };

  const setQrData = (data: string) => {
    setSettings(prev => ({ ...prev, qrData: data }));
  };

  const handlePrint = async () => {
    if (!qrData) {
      alert("Please enter QR code data");
      return;
    }

    try {
      const response = await fetch("/api/print_qr", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          printer: selectedPrinter.id,
          model: selectedPrinter.name,
          label_size: selectedLabelSize,
          data: qrData
        }),
      });

      if (response.ok) {
        alert("QR label printed successfully!");
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
    } catch (err) {
      console.error("Print failed:", err);
      
      // Check if it's a USB device error
      const errorMessage = err instanceof Error ? err.message : JSON.stringify(err);
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
        alert("Print failed - check console for details");
      }
    }
  };

  const handleResetSettings = () => {
    if (confirm("Reset all settings to defaults?")) {
      setSettings({
        selectedPrinter: DEFAULT_PRINTER,
        selectedLabelSize: DEFAULT_LABEL_SIZE,
        qrData: ""
      });
    }
  };

  return (
    <div className="container mx-auto p-4 max-w-2xl">
      {/* Printer Disconnection Notification */}
      {showPrinterDisconnected && printerError && (
        <PrinterDisconnectedNotification
          error={printerError}
          onRetry={handlePrinterRetry}
          onCancel={handlePrinterCancel}
          isRetrying={isRecovering}
        />
      )}
      
      <h1 className="text-2xl font-bold mb-6">QR Label Printer</h1>
      
      <div className="space-y-6">
        <QRCodeGenerator 
          qrData={qrData}
          onQrDataChange={setQrData}
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
              
              <div>
                <label className="block mb-2">Printer</label>
                <PrinterSelector
                  value={selectedPrinter.id}
                  onSelectPrinter={setSelectedPrinter}
                  onDetectLabelSize={(labelSizeId) => {
                    if (settingsMode === "auto") {
                      console.log("Auto-detected label size:", labelSizeId);
                      setSettings(prev => ({ ...prev, selectedLabelSize: labelSizeId }));
                    }
                  }}
                  manualOverride={settingsMode === "manual"}
                />
              </div>
              
              {/* Auto Mode Section */}
              {settingsMode === "auto" && (
                <div className="border border-blue-200 rounded-lg p-4 bg-blue-50">
                  <h4 className="text-sm font-medium mb-2 text-blue-800">Auto Detection</h4>
                  <p className="text-xs text-blue-600 mb-3">
                    Label size is automatically detected from printer and tape.
                  </p>
                  <div className="space-y-3">
                    <div>
                      <label className="block text-xs mb-1">Detected Label Size</label>
                      <div className="text-sm p-2 bg-white rounded border">
                        {selectedLabelSize || "Detecting..."}
                      </div>
                    </div>
                  </div>
                </div>
              )}
              
              {/* Manual Mode Section */}
              {settingsMode === "manual" && (
                <div className="border border-gray-200 rounded-lg p-4 bg-gray-50">
                  <h4 className="text-sm font-medium mb-2 text-gray-800">Manual Settings</h4>
                  <p className="text-xs text-gray-600 mb-3">
                    Manually select label size.
                  </p>
                  <div>
                    <label className="block mb-2">Label Size</label>
                    <LabelSizeSelector
                      value={selectedLabelSize}
                      onLabelSizeChange={setSelectedLabelSize}
                    />
                  </div>
                </div>
              )}
            </AccordionContent>
          </AccordionItem>
        </Accordion>

        <div className="space-y-2">
          <Button 
            onClick={handlePrint}
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
    </div>
  );
}
