import React from 'react';
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Button } from "@/components/ui/button";
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
  // Optional text-internal alignment (left/center/right within the text block)
  onTextAlignChange?: (value: 'left' | 'center' | 'right') => void;
  textAlignValue?: 'left' | 'center' | 'right';
}

const TextAlignmentSelector: React.FC<TextAlignmentSelectorProps> = ({
  onHorizontalChange,
  onVerticalChange,
  horizontalValue,
  verticalValue,
  onTextRotationChange,
  textRotationValue,
  onTextAlignChange,
  textAlignValue,
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

  const handleTextAlignChange = (value: 'left' | 'center' | 'right' | '') => {
    if (onTextAlignChange) onTextAlignChange(value || 'left');
  };

  return (
    <div className="flex flex-wrap gap-1 items-center">
      {onTextAlignChange && (
        <ToggleGroup type="single" onValueChange={handleTextAlignChange} value={textAlignValue ?? 'left'} variant="outline" size="sm">
          <ToggleGroupItem value="left" aria-label="Text align left" title="Text align left">
            <AlignLeft className="h-4 w-4" />
          </ToggleGroupItem>
          <ToggleGroupItem value="center" aria-label="Text align center" title="Text align center">
            <AlignCenter className="h-4 w-4" />
          </ToggleGroupItem>
          <ToggleGroupItem value="right" aria-label="Text align right" title="Text align right">
            <AlignRight className="h-4 w-4" />
          </ToggleGroupItem>
        </ToggleGroup>
      )}
      <ToggleGroup type="single" onValueChange={handleHorizontalChange} value={horizontalValue} variant="outline" size="sm">
        <ToggleGroupItem value="start" aria-label="Block align left" title="Block align left">
          <AlignVerticalJustifyEnd className="h-4 w-4 rotate-90" />
        </ToggleGroupItem>
        <ToggleGroupItem value="center" aria-label="Block align center" title="Block align center">
          <AlignVerticalJustifyCenter className="h-4 w-4 rotate-90" />
        </ToggleGroupItem>
        <ToggleGroupItem value="end" aria-label="Block align right" title="Block align right">
          <AlignVerticalJustifyStart className="h-4 w-4 rotate-90" />
        </ToggleGroupItem>
      </ToggleGroup>
      <ToggleGroup type="single" onValueChange={handleVerticalChange} value={verticalValue} variant="outline" size="sm">
        <ToggleGroupItem value="start" aria-label="Align top" title="Align top">
          <AlignVerticalJustifyStart className="h-4 w-4" />
        </ToggleGroupItem>
        <ToggleGroupItem value="center" aria-label="Align middle" title="Align middle">
          <AlignVerticalJustifyCenter className="h-4 w-4" />
        </ToggleGroupItem>
        <ToggleGroupItem value="end" aria-label="Align bottom" title="Align bottom">
          <AlignVerticalJustifyEnd className="h-4 w-4" />
        </ToggleGroupItem>
      </ToggleGroup>
      <Button variant="outline" onClick={handleRotateClick} size="icon" className="h-8 w-8" title={`Rotation: ${textRotationValue}°`}>
        <RotateCw className={`h-4 w-4 transition-transform duration-200 ${
          textRotationValue === 90 ? 'transform rotate-90' :
          textRotationValue === 270 ? 'transform -rotate-90' : ''
        }`} />
      </Button>
    </div>
  );
};

export default TextAlignmentSelector;
