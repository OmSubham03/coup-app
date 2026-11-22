function applyHomeTheme(theme) {
  const dark = theme === 'dark';
  document.getElementById('screen-menu').dataset.homeTheme = dark ? 'dark' : 'light';
  for (const id of ['home-theme-toggle', 'lobby-theme-toggle', 'settings-theme-toggle']) {
    const toggle = document.getElementById(id);
    if (!toggle) continue;
    toggle.setAttribute('aria-pressed', String(dark));
    toggle.title = dark ? 'Switch to white theme' : 'Switch to dark blue theme';
  }
}

function toggleHomeTheme() {
  setPlayerTheme(document.getElementById('screen-menu').dataset.homeTheme !== 'dark');
}

function setPlayerTheme(dark) {
  const theme = dark ? 'dark' : 'light';
  applyHomeTheme(theme);
  try {
    localStorage.setItem('timecrusher_home_theme', theme);
  } catch (error) {
    console.error('Unable to save theme:', error);
    for (const id of ['home-search-status', 'game-settings-status', 'invite-status', 'guide-status']) {
      const status = document.getElementById(id);
      if (status) status.textContent = 'Theme changed, but your browser could not save it for your next visit.';
    }
  }
}

function initHomeTheme() {
  let theme = 'light';
  try {
    theme = localStorage.getItem('timecrusher_home_theme');
  } catch (error) {
    console.error('Unable to load homepage theme:', error);
    const status = document.getElementById('home-search-status') || document.getElementById('guide-status');
    if (status) status.textContent = 'Your saved theme could not be loaded. The white theme is available.';
  }
  applyHomeTheme(theme);
}
