import { Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";

export function DarkModeToggle() {
  const { theme, setTheme } = useTheme();

  const toggleTheme = () => {
    setTheme(theme === "dark" ? "light" : "dark");
  };

  return (
    <button
      type="button"
      onClick={toggleTheme}
      className="inline-flex items-center justify-center h-6 w-6 rounded-md hover:bg-accent transition-colors"
      title="Toggle dark mode"
    >
      <Sun className="h-3.5 w-3.5 text-muted-foreground block dark:hidden" />
      <Moon className="h-3.5 w-3.5 text-muted-foreground hidden dark:block" />
      <span className="sr-only">Toggle theme</span>
    </button>
  );
}
