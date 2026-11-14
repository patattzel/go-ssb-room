// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

const invitation = document.getElementById('claim-invite-uri');
const status = document.getElementById('waiting');

invitation.addEventListener('click', () => {
  status.hidden = false;
});
