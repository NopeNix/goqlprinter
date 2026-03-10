import { Textarea } from './ui/textarea';
import { Label } from './ui/label';
import { Slider } from './ui/slider';

interface QRCodeGeneratorProps {
  qrData: string;
  onQrDataChange: (data: string) => void;
  qrScale: number[];
  onQrScaleChange: (scale: number[]) => void;
}

const QRCodeGenerator = ({ qrData, onQrDataChange, qrScale, onQrScaleChange }: QRCodeGeneratorProps) => {
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
        <div>
          <Label htmlFor="qr-scale">QR Scale: {qrScale[0]}%</Label>
          <Slider
            id="qr-scale"
            min={10}
            max={200}
            step={1}
            value={qrScale}
            onValueChange={onQrScaleChange}
          />
        </div>
      )}
    </div>
  );
};

export default QRCodeGenerator;
