// SPDX-FileCopyrightText: 2026 The go-ssb-room Cloudron packaging contributors
//
// SPDX-License-Identifier: MIT

for (const button of document.querySelectorAll('[data-copy-field]')) {
  const field = document.getElementById(button.dataset.copyField);
  const status = document.getElementById(button.getAttribute('aria-describedby'));
  if (!field || !status) continue;

  button.hidden = false;
  button.addEventListener('click', async () => {
    try {
      await navigator.clipboard.writeText(field.value);
      status.textContent = button.dataset.copied;
    } catch {
      field.focus();
      field.select();
      status.textContent = button.dataset.select;
    }
  });
}
