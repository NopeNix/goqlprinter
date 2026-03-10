export const saveSettings = <T>(settings: T) => {
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
