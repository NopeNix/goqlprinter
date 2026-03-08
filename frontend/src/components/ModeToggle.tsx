import React from "react";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Label } from "@/components/ui/label";

interface ModeToggleProps {
  value: "auto" | "manual";
  onValueChange: (value: "auto" | "manual") => void;
  disabled?: boolean;
}

const ModeToggle: React.FC<ModeToggleProps> = ({ 
  value, 
  onValueChange,
  disabled = false
}) => {
  return (
    <div className="space-y-2">
      <Label>Settings Mode</Label>
      <ToggleGroup 
        type="single" 
        value={value} 
        onValueChange={onValueChange}
        disabled={disabled}
        className="w-full"
      >
        <ToggleGroupItem 
          value="auto" 
          className="flex-1 py-2 text-sm font-medium data-[state=on]:bg-blue-500 data-[state=on]:text-white"
        >
          Auto Detection
        </ToggleGroupItem>
        <ToggleGroupItem 
          value="manual" 
          className="flex-1 py-2 text-sm font-medium data-[state=on]:bg-blue-500 data-[state=on]:text-white"
        >
          Manual Override
        </ToggleGroupItem>
      </ToggleGroup>
      <p className="text-xs text-muted-foreground">
        {value === "auto" 
          ? "Settings automatically detected from printer and tape" 
          : "Manually select label size and orientation"}
      </p>
    </div>
  );
};

export default ModeToggle;