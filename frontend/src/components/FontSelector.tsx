import React, { useEffect, useRef, useState } from 'react';
import { Check, ChevronsUpDown } from 'lucide-react';
import { fontApi } from '../api/endpoints';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';

interface FontSelectorProps {
  onSelectFont: (fontFamily: string) => void;
}

const FontSelector: React.FC<FontSelectorProps> = ({ onSelectFont }) => {
  const [fonts, setFonts] = useState<string[]>([]);
  const [selectedFont, setSelectedFont] = useState<string | undefined>(undefined);
  const [open, setOpen] = useState(false);
  const onSelectFontRef = useRef(onSelectFont);
  onSelectFontRef.current = onSelectFont;

  useEffect(() => {
    fontApi.list()
      .then(data => {
        setFonts(data.fonts);
        if (data.fonts.length > 0) {
          const initialFont = data.fonts[0];
          setSelectedFont(initialFont);
          onSelectFontRef.current(initialFont);
        }
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    fonts.forEach(font => {
      const fontName = font.replace(/ /g, '+');
      const link = document.createElement('link');
      link.href = `https://fonts.googleapis.com/css2?family=${fontName}`;
      link.rel = 'stylesheet';
      document.head.appendChild(link);
    });
  }, [fonts]);

  const handleSelect = (value: string) => {
    setSelectedFont(value);
    onSelectFont(value);
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between"
          style={{ fontFamily: selectedFont }}
        >
          {selectedFont || "Select a font..."}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0">
        <Command>
          <CommandInput placeholder="Search fonts..." />
          <CommandList>
            <CommandEmpty>No font found.</CommandEmpty>
            <CommandGroup>
              {fonts.map(font => (
                <CommandItem
                  key={font}
                  value={font}
                  onSelect={handleSelect}
                  style={{ fontFamily: font }}
                >
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4",
                      selectedFont === font ? "opacity-100" : "opacity-0"
                    )}
                  />
                  {font}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
};

export default FontSelector;
