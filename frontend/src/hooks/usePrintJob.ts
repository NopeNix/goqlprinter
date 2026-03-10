import { useState, useCallback } from "react";
import { toast } from "sonner";
import usePrinterRecovery from "./usePrinterRecovery";
import type { LabelSettings } from "./useLabelSettings";
import type { PrinterInfo } from "./usePrinterStatus";
import { ApiError } from "../api/client";
import { printApi } from "../api/endpoints";

export interface UsePrintJobOptions {
  settings: LabelSettings;
  svgData: string | null;
  pngFile: File | null;
  onPrinterRecovered?: (printer: PrinterInfo) => void;
}

export interface UsePrintJobReturn {
  handlePrint: () => Promise<void>;
  printerError: string | null;
  showPrinterDisconnected: boolean;
  isRecovering: boolean;
  handlePrinterRetry: () => void;
  handlePrinterCancel: () => void;
}

function isUsbDeviceError(error: unknown): boolean {
  if (error instanceof ApiError && error.code === "USB_DEVICE_NOT_FOUND") {
    return true;
  }
  const message =
    error instanceof Error ? error.message : String(error);
  return (
    message.includes("USB device not found") ||
    message.includes("device not found")
  );
}

export function usePrintJob({
  settings,
  svgData,
  pngFile,
  onPrinterRecovered,
}: UsePrintJobOptions): UsePrintJobReturn {
  const [printerError, setPrinterError] = useState<string | null>(null);
  const [showPrinterDisconnected, setShowPrinterDisconnected] = useState(false);

  const {
    isRecovering,
    startBackgroundRecovery,
    stopRecovery,
    manualRetryNow,
  } = usePrinterRecovery();

  const handlePrinterRecovered = useCallback(
    (printer: PrinterInfo) => {
      setPrinterError(null);
      setShowPrinterDisconnected(false);
      onPrinterRecovered?.(printer);
    },
    [onPrinterRecovered],
  );

  const handlePrinterRecoveryFailed = useCallback(() => {
    // Recovery attempts exhausted, keep showing the notification for user action
  }, []);

  const handlePrinterRetry = useCallback(() => {
    manualRetryNow(
      (error: string) => {
        setPrinterError(error);
      },
      handlePrinterRecovered,
    );
  }, [manualRetryNow, handlePrinterRecovered]);

  const handlePrinterCancel = useCallback(() => {
    stopRecovery();
    setShowPrinterDisconnected(false);
    setPrinterError(null);
  }, [stopRecovery]);

  const triggerRecovery = useCallback(
    (errorMessage: string) => {
      setPrinterError(errorMessage);
      setShowPrinterDisconnected(true);
      startBackgroundRecovery(
        (error: string) => {
          setPrinterError(error);
        },
        handlePrinterRecovered,
        handlePrinterRecoveryFailed,
      );
    },
    [startBackgroundRecovery, handlePrinterRecovered, handlePrinterRecoveryFailed],
  );

  const handlePrint = useCallback(async () => {
    if (!settings.selectedPrinter?.id || !settings.selectedLabelSize) {
      toast.error("Please select a printer and label size.");
      return;
    }

    let endpoint = "";
    const customHeight =
      settings.heightMode === "manual" ? settings.customHeightMM : 0;
    let payload: Record<string, unknown> = {
      printer: settings.selectedPrinter.id,
      model: settings.selectedPrinter.name,
      label_size: settings.selectedLabelSize,
      ...(customHeight > 0 && { custom_height_mm: customHeight }),
    };

    if (settings.printMode === "text") {
      if (!settings.labelText || !settings.selectedFont) {
        toast.error("Please enter text and select a font.");
        return;
      }
      endpoint = "/api/print";
      payload = {
        ...payload,
        text: settings.labelText,
        font_family: settings.selectedFont,
        font_size: settings.fontSize[0],
        orientation: settings.selectedOrientation,
        horizontal_alignment: settings.horizontalAlignment,
        vertical_alignment: settings.verticalAlignment,
        text_rotation: settings.textRotation,
      };
    } else if (settings.printMode === "qr") {
      if (!settings.qrData) {
        toast.error("Please enter data for the QR code.");
        return;
      }
      endpoint = "/api/print_qr";
      payload = {
        ...payload,
        data: settings.qrData,
        qr_scale: settings.qrScale[0] / 100,
        horizontal_alignment: settings.horizontalAlignment,
        vertical_alignment: settings.verticalAlignment,
      };
    } else if (settings.printMode === "svg") {
      if (!svgData) {
        toast.error("Please load an SVG file.");
        return;
      }
      endpoint = "/api/print_svg";
      payload = {
        ...payload,
        svg_data: svgData,
        orientation: settings.selectedOrientation,
        svg_scale: settings.svgScale[0] / 100,
        horizontal_alignment: settings.horizontalAlignment,
        vertical_alignment: settings.verticalAlignment,
      };
    } else if (settings.printMode === "png") {
      if (!pngFile) {
        toast.error("Please select a PNG file.");
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
        toast.error("Could not process PNG file.");
        return;
      }
    }

    try {
      const printFn =
        endpoint === "/api/print"
          ? printApi.text
          : endpoint === "/api/print_qr"
            ? printApi.qr
            : endpoint === "/api/print_svg"
              ? printApi.svg
              : printApi.png;

      await printFn(payload);
      toast.success("Print job sent successfully!");
    } catch (error) {
      if (isUsbDeviceError(error)) {
        const errorMessage =
          error instanceof Error ? error.message : String(error);
        triggerRecovery(errorMessage);
      } else if (error instanceof ApiError) {
        toast.error(error.message);
      } else {
        toast.error("Failed to send print job.");
      }
    }
  }, [settings, svgData, pngFile, triggerRecovery]);

  return {
    handlePrint,
    printerError,
    showPrinterDisconnected,
    isRecovering,
    handlePrinterRetry,
    handlePrinterCancel,
  };
}

export default usePrintJob;
