import React, { useEffect, useState } from 'react';
import { labelApi } from '../api/endpoints';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select';

interface LabelSize {
  id: string;
  name: string;
}

interface LabelSizeSelectorProps {
  value?: string;
  onLabelSizeChange: (labelSize: LabelSize) => void;
}

const LabelSizeSelector: React.FC<LabelSizeSelectorProps> = ({ value, onLabelSizeChange }) => {
  const [labelSizes, setLabelSizes] = useState<LabelSize[]>([]);

  useEffect(() => {
    labelApi.sizes()
      .then(data => {
        const sizes: LabelSize[] = data.label_sizes;
        setLabelSizes(sizes);
        // Set a default value only if no value is provided from the parent
        if (!value && sizes.length > 0) {
          onLabelSizeChange(sizes[0]);
        }
      })
      .catch(() => {});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Run only once on mount

  const handleValueChange = (id: string) => {
    const selected = labelSizes.find(size => size.id === id);
    if (selected) {
      onLabelSizeChange(selected);
    }
  };

  return (
    <Select value={value} onValueChange={handleValueChange}>
      <SelectTrigger className="w-full h-9 text-sm">
        <SelectValue placeholder="Select label size" />
      </SelectTrigger>
      <SelectContent>
        {labelSizes.map(labelSize => (
          <SelectItem key={labelSize.id} value={labelSize.id}>
            {labelSize.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};

export default LabelSizeSelector;
