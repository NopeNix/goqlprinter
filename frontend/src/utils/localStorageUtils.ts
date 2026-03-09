export const saveSettings = (settings: Record<string, any>) => {
  try {
    localStorage.setItem('labelSettings', JSON.stringify(settings));
  } catch {
    // ignore localStorage errors
  }
};

export const loadSettings = () => {
  try {
    const saved = localStorage.getItem('labelSettings');
    return saved ? JSON.parse(saved) : null;
  } catch {
    return null;
  }
};
