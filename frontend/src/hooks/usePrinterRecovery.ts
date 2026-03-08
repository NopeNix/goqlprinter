import { useState, useCallback, useRef } from 'react';

export interface PrinterInfo {
  id: string;
  name: string;
}

export const FILE_PRINTER = { id: "file", name: "Print to File (debug)" };
export const STORAGE_KEY = "selectedPrinter";
export const PRINTER_NAME_KEY = "selectedPrinterName";

const RECOVERY_INTERVAL = 5000; // 5 seconds between recovery attempts
const MAX_RECOVERY_ATTEMPTS = 12; // Maximum 1 minute of recovery attempts (12 * 5s)

export const usePrinterRecovery = () => {
  const [isRecovering, setIsRecovering] = useState(false);
  const [recoveryAttempts, setRecoveryAttempts] = useState(0);
  const [lastError, setLastError] = useState<string | null>(null);
  const recoveryTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const clearRecoveryTimeout = useCallback(() => {
    if (recoveryTimeoutRef.current) {
      clearTimeout(recoveryTimeoutRef.current);
      recoveryTimeoutRef.current = null;
    }
  }, []);

  const loadSavedPrinter = useCallback((): { id: string | null; name: string | null } => {
    try {
      const savedId = localStorage.getItem(STORAGE_KEY);
      const savedName = localStorage.getItem(PRINTER_NAME_KEY);
      return {
        id: savedId ? JSON.parse(savedId) : null,
        name: savedName ? JSON.parse(savedName) : null
      };
    } catch {
      return { id: null, name: null };
    }
  }, []);

  const savePrinter = useCallback((printerId: string, printerName?: string) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(printerId));
      if (printerName) {
        localStorage.setItem(PRINTER_NAME_KEY, JSON.stringify(printerName));
      }
    } catch {
      // Silent fail if localStorage is not available
    }
  }, []);

  const checkPrinterAvailable = useCallback(async (printerId: string): Promise<boolean> => {
    try {
      const response = await fetch('/api/printers');
      if (!response.ok) return false;
      
      const data = await response.json();
      const printersList = data.printers || [];
      
      return printersList.some((p: PrinterInfo) => p.id === printerId);
    } catch {
      return false;
    }
  }, []);

  const attemptPrinterRecovery = useCallback(async (
    onRecovered?: (printer: PrinterInfo) => void,
    onFailed?: () => void
  ): Promise<boolean> => {
    if (isRecovering) return false;

    setIsRecovering(true);
    setRecoveryAttempts(prev => prev + 1);

    try {
      const savedPrinter = loadSavedPrinter();
      const { id: savedPrinterId, name: savedPrinterName } = savedPrinter;
      
      if (!savedPrinterId || savedPrinterId === FILE_PRINTER.id) {
        setIsRecovering(false);
        return false;
      }

      // Get available printers first to check by both ID and name
      const response = await fetch('/api/printers');
      if (!response.ok) {
        setIsRecovering(false);
        return false;
      }
      
      const data = await response.json();
      const printersList = data.printers || [];
      
      // First try to find by exact ID
      const exactPrinter = printersList.find((p: PrinterInfo) => p.id === savedPrinterId);
      
      if (exactPrinter && onRecovered) {
        onRecovered(exactPrinter);
        setIsRecovering(false);
        setRecoveryAttempts(0);
        setLastError(null);
        return true;
      }
      
      // If not found by ID, try to find by name (for USB address changes)
      if (savedPrinterName && savedPrinterName !== FILE_PRINTER.name) {
        const printerByName = printersList.find((p: PrinterInfo) => 
          p.name === savedPrinterName && p.id !== FILE_PRINTER.id
        );
        
        if (printerByName && onRecovered) {
          // Update the saved printer ID to the new address but keep the same name
          savePrinter(printerByName.id, printerByName.name);
          onRecovered(printerByName);
          setIsRecovering(false);
          setRecoveryAttempts(0);
          setLastError(null);
          return true;
        }
      }

      // Printer still not available, check if we should continue trying
      if (recoveryAttempts >= MAX_RECOVERY_ATTEMPTS) {
        setIsRecovering(false);
        setRecoveryAttempts(0);
        if (onFailed) onFailed();
        return false;
      }

      // Schedule next recovery attempt
      recoveryTimeoutRef.current = setTimeout(() => {
        attemptPrinterRecovery(onRecovered, onFailed);
      }, RECOVERY_INTERVAL);

      return false;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      setLastError(errorMessage);
      
      // Continue trying if we haven't maxed out attempts
      if (recoveryAttempts < MAX_RECOVERY_ATTEMPTS) {
        recoveryTimeoutRef.current = setTimeout(() => {
          attemptPrinterRecovery(onRecovered, onFailed);
        }, RECOVERY_INTERVAL);
      } else {
        setIsRecovering(false);
        setRecoveryAttempts(0);
        if (onFailed) onFailed();
      }
      
      return false;
    }
  }, [isRecovering, recoveryAttempts, loadSavedPrinter, checkPrinterAvailable, savePrinter]);

  const startBackgroundRecovery = useCallback((
    _onError: (error: string) => void,
    onRecovered: (printer: PrinterInfo) => void,
    onMaxAttemptsReached?: () => void
  ) => {
    clearRecoveryTimeout();
    setRecoveryAttempts(0);
    
    attemptPrinterRecovery(
      onRecovered,
      () => {
        if (onMaxAttemptsReached) {
          onMaxAttemptsReached();
        }
      }
    );
  }, [clearRecoveryTimeout, attemptPrinterRecovery]);

  const stopRecovery = useCallback(() => {
    clearRecoveryTimeout();
    setIsRecovering(false);
    setRecoveryAttempts(0);
    setLastError(null);
  }, [clearRecoveryTimeout]);

  const manualRetryNow = useCallback(async (
    onError: (error: string) => void,
    onRecovered: (printer: PrinterInfo) => void
  ) => {
    clearRecoveryTimeout();
    setIsRecovering(true);
    
    const success = await attemptPrinterRecovery(onRecovered);
    
    if (!success && lastError) {
      onError(lastError);
    }
    
    if (!success) {
      setIsRecovering(false);
    }
  }, [clearRecoveryTimeout, attemptPrinterRecovery, lastError]);

  // Cleanup on unmount
  useState(() => {
    return () => {
      clearRecoveryTimeout();
    };
  });

  return {
    isRecovering,
    recoveryAttempts,
    lastError,
    startBackgroundRecovery,
    stopRecovery,
    manualRetryNow,
    savePrinter,
    loadSavedPrinter,
    checkPrinterAvailable,
    RECOVERY_INTERVAL,
    MAX_RECOVERY_ATTEMPTS
  };
};

export default usePrinterRecovery;