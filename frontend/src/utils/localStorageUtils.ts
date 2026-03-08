export const saveSettings = (settings: Record<string, any>) => {
  try {
    localStorage.setItem('labelSettings', JSON.stringify(settings));
  } catch (error) {
    console.error('Failed to save settings:', error);
  }
};

export const loadSettings = () => {
  try {
    const saved = localStorage.getItem('labelSettings');
    return saved ? JSON.parse(saved) : null;
  } catch (error) {
    console.error('Failed to load settings:', error);
    return null;
  }
};
