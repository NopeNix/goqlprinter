import { QRCodeCanvas } from 'qrcode.react';
import { Textarea } from './ui/textarea';
import { Label } from './ui/label';

interface QRCodeGeneratorProps {
  qrData: string;
  onQrDataChange: (data: string) => void;
}

const QRCodeGenerator = ({ qrData, onQrDataChange }: QRCodeGeneratorProps) => {
  return (
    <div className="space-y-4">
      <div>
        <Label htmlFor="qr-data">QR Code Data</Label>
        <Textarea
          id="qr-data"
          value={qrData}
          onChange={(e) => onQrDataChange(e.target.value)}
          placeholder="Enter data for QR code (e.g., URL)"
          rows={4}
        />
      </div>
      {qrData && (
        <div className="flex items-center justify-center">
          <QRCodeCanvas value={qrData} size={128} />
        </div>
      )}
    </div>
  );
};

export default QRCodeGenerator;
