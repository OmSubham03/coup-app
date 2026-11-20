// Preset IDs are persisted locally and sent over the wire; keep them stable.
const avatarPresets = new Map([
  ['fox', '🦊'], ['panda', '🐼'], ['owl', '🦉'],
  ['tiger', '🐯'], ['koala', '🐨'], ['frog', '🐸'],
  ['penguin', '🐧'], ['octopus', '🐙'], ['unicorn', '🦄']
]);
let roomAvatars = new Map();

function savedAvatar() {
  const id = localStorage.getItem('tc_avatar');
  return avatarPresets.has(id) ? id : '';
}

function avatarInitials(name) {
  const words = (name || '').replace(/#\d{4}$/, '').trim().split(/\s+/);
  return words.slice(0, 2).map(word => Array.from(word)[0] || '').join('').toUpperCase() || '?';
}

function avatarFace(player) {
  return avatarPresets.get(roomAvatars.get(player.id)) || esc(avatarInitials(player.name));
}

function playerBadge(player) {
  return `<span class="player-avatar-badge" aria-hidden="true">${avatarFace(player)}</span>`;
}

function renderAvatarPicker() {
  const selected = savedAvatar();
  document.getElementById('avatar-options').innerHTML = [['', 'Initials'], ...avatarPresets].map(([id, face]) =>
    `<label><input type="radio" name="profile-avatar" value="${id}" ${id === selected ? 'checked' : ''}>
     <span>${id ? `<b aria-hidden="true">${face}</b>${id}` : '<b id="avatar-initials-preview" aria-hidden="true"></b>Initials'}</span></label>`
  ).join('');
  previewAvatarInitials();
}

function previewAvatarInitials() {
  document.getElementById('avatar-initials-preview').textContent = avatarInitials(document.getElementById('popup-name-input').value);
}
