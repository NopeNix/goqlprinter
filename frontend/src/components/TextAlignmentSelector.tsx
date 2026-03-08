import React from 'react';
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { AlignLeft, AlignCenter, AlignRight, AlignVerticalJustifyStart, AlignVerticalJustifyCenter, AlignVerticalJustifyEnd, RotateCw } from 'lucide-react';

interface TextAlignmentSelectorProps {
  onHorizontalChange: (value: 'start' | 'center' | 'end') => void;
  onVerticalChange: (value: 'start' | 'center' | 'end') => void;
  horizontalValue: 'start' | 'center' | 'end';
  verticalValue: 'start' | 'center' | 'end';
  onTextRotationChange: (value: number) => void;
  textRotationValue: number;
  onOrientationChange: (value: 'standard' | 'rotated') => void;
  orientationValue: 'standard' | 'rotated';
}

const TextAlignmentSelector: React.FC<TextAlignmentSelectorProps> = ({ 
  onHorizontalChange, 
  onVerticalChange, 
  horizontalValue, 
  verticalValue,
  onTextRotationChange,
  textRotationValue
}) => {
  const handleHorizontalChange = (value: 'start' | 'center' | 'end' | '') => {
    // If the user deselects the toggle, default to 'center'
    onHorizontalChange(value || 'center');
  };

  const handleVerticalChange = (value: 'start' | 'center' | 'end' | '') => {
    // If the user deselects the toggle, default to 'center'
    onVerticalChange(value || 'center');
  };

  const handleRotateClick = () => {
    // Cycle through 0, 90, 270
    if (textRotationValue === 0) onTextRotationChange(90);
    else if (textRotationValue === 90) onTextRotationChange(270);
    else onTextRotationChange(0);
  };

  return (
    <div className="flex gap-4 items-end">
      <div>
        <Label className="mb-2 block">Vaakatasaus</Label>
        <ToggleGroup type="single" onValueChange={handleHorizontalChange} value={horizontalValue} variant="outline">
          <ToggleGroupItem value="start" aria-label="Align left">
            <AlignLeft className="h-4 w-4" />
          </ToggleGroupItem>
          <ToggleGroupItem value="center" aria-label="Align center">
            <AlignCenter className="h-4 w-4" />
          </ToggleGroupItem>
          <ToggleGroupItem value="end" aria-label="Align right">
            <AlignRight className="h-4 w-4" />
          </ToggleGroupItem>
        </ToggleGroup>
      </div>
      <div>
        <Label className="mb-2 block">Pystytasaus</Label>
        <ToggleGroup type="single" onValueChange={handleVerticalChange} value={verticalValue} variant="outline">
          <ToggleGroupItem value="start" aria-label="Align top">
            <AlignVerticalJustifyStart className="h-4 w-4" />
          </ToggleGroupItem>
          <ToggleGroupItem value="center" aria-label="Align middle">
            <AlignVerticalJustifyCenter className="h-4 w-4" />
          </ToggleGroupItem>
          <ToggleGroupItem value="end" aria-label="Align bottom">
            <AlignVerticalJustifyEnd className="h-4 w-4" />
          </ToggleGroupItem>
        </ToggleGroup>
      </div>
      <div>
        <Label className="mb-2 block">Kierto</Label>
        <Button variant="outline" onClick={handleRotateClick} size="icon">
          <RotateCw className={`h-4 w-4 transition-transform duration-200 ${
            textRotationValue === 90 ? 'transform rotate-90' : 
            textRotationValue === 270 ? 'transform -rotate-90' : ''
          }`} />
        </Button>
      </div>
    </div>
  );
};

export default TextAlignmentSelector;
