import React, { useState, useCallback } from 'react';
import { Alert, AlertDescription } from './ui/alert';
import { Button } from './ui/button';
import { RefreshCw, AlertCircle, CheckCircle, X } from 'lucide-react';

interface PrinterDisconnectedNotificationProps {
  error: string;
  onRetry: () => void;
  onCancel: () => void;
  isRetrying: boolean;
}

const PrinterDisconnectedNotification: React.FC<PrinterDisconnectedNotificationProps> = ({
  error,
  onRetry,
  onCancel,
  isRetrying
}) => {
  const [isCancelled, setIsCancelled] = useState(false);

  const handleCancel = useCallback(() => {
    setIsCancelled(true);
    onCancel();
  }, [onCancel]);

  const handleRetry = useCallback(() => {
    setIsCancelled(false);
    onRetry();
  }, [onRetry]);

  if (isCancelled) {
    return null;
  }

  const isUSBBusError = error.includes('USB device not found');
  const message = isUSBBusError 
    ? 'Please connect the printer or turn it on'
    : 'Printer connection failed. Please check the printer status.';

  return (
    <Alert variant="destructive" className="mb-4">
      <div className="flex items-start space-x-2">
        <div className="flex-shrink-0">
          <AlertCircle className="h-4 w-4" />
        </div>
        <div className="flex-1 space-y-2">
          <AlertDescription className="font-medium">
            {message}
          </AlertDescription>
          <div className="flex items-center space-x-2 text-sm text-muted-foreground">
            <span className="break-all">{error}</span>
          </div>
          <div className="flex space-x-2 mt-3">
            <Button
              size="sm"
              onClick={handleRetry}
              disabled={isRetrying}
              variant="outline"
              className="flex items-center space-x-1"
            >
              {isRetrying ? (
                <RefreshCw className="h-3 w-3 animate-spin" />
              ) : (
                <CheckCircle className="h-3 w-3" />
              )}
              <span>Printer is now available</span>
            </Button>
            <Button
              size="sm"
              onClick={handleCancel}
              variant="outline"
              className="flex items-center space-x-1"
            >
              <X className="h-3 w-3" />
              <span>Cancel</span>
            </Button>
          </div>
        </div>
      </div>
    </Alert>
  );
};

export default PrinterDisconnectedNotification;