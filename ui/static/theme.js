// @ts-check
const html = document.documentElement;
const btn = document.getElementById('theme-toggle');
if (!btn) throw new Error('missing #theme-toggle');
const prefersDark = () => matchMedia('(prefers-color-scheme: dark)').matches;

/** @param {string} theme */
const apply = (theme) => {
  html.dataset.theme = theme;
  btn.textContent = theme === 'dark' ? '☀️' : '🌙';
};

const saved = localStorage.getItem('theme');
if (saved) {
  apply(saved);
} else {
  btn.textContent = prefersDark() ? '☀️' : '🌙';
}

btn.addEventListener('click', () => {
  const current = html.dataset.theme;
  const next = current
    ? (current === 'dark' ? 'light' : 'dark')
    : (prefersDark() ? 'light' : 'dark');

  apply(next);
  localStorage.setItem('theme', next);
});
