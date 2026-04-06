// ========== VOICE CHAT: WebSocket relay audio ==========
// Sends raw PCM audio over the existing WebSocket connection.
// No WebRTC/TURN needed — works through any firewall.

let voiceMicOn = false;
let voiceSpeakerOn = true;
let localStream = null;
let captureCtx = null;
let captureProcessor = null;

const TARGET_SAMPLE_RATE = 8000;
const SILENCE_THRESHOLD = 0.005; // RMS below this = silence, don't send

// Single shared playback AudioContext (created on first user gesture)
let playbackCtx = null;
const peerNextTime = {}; // peerId -> next scheduled play time

function ensurePlaybackCtx() {
  if (!playbackCtx || playbackCtx.state === 'closed') {
    playbackCtx = new (window.AudioContext || window.webkitAudioContext)({ sampleRate: TARGET_SAMPLE_RATE });
  }
  if (playbackCtx.state === 'suspended') {
    playbackCtx.resume();
  }
  return playbackCtx;
}

// ---- Mic toggle ----
async function toggleMic() {
  ensurePlaybackCtx(); // user gesture — ensure playback ctx is alive
  if (voiceMicOn) {
    stopMic();
  } else {
    await startMic();
  }
  updateVoiceUI();
}

async function startMic() {
  try {
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      alert('Your browser does not support microphone access. Try using Chrome.');
      return;
    }
    localStream = await navigator.mediaDevices.getUserMedia({
      audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
      video: false
    });
    voiceMicOn = true;

    captureCtx = new AudioContext();
    const source = captureCtx.createMediaStreamSource(localStream);
    captureProcessor = captureCtx.createScriptProcessor(4096, 1, 1);

    const nativeSR = captureCtx.sampleRate;
    const ratio = Math.round(nativeSR / TARGET_SAMPLE_RATE);

    captureProcessor.onaudioprocess = (e) => {
      if (!voiceMicOn) return;
      const input = e.inputBuffer.getChannelData(0);

      // Voice activity detection (RMS)
      let sum = 0;
      for (let i = 0; i < input.length; i++) sum += input[i] * input[i];
      const rms = Math.sqrt(sum / input.length);
      if (rms < SILENCE_THRESHOLD) return;

      // Downsample to target rate
      const downLen = Math.floor(input.length / ratio);
      const int16 = new Int16Array(downLen);
      for (let i = 0; i < downLen; i++) {
        const s = input[i * ratio];
        int16[i] = Math.max(-32768, Math.min(32767, Math.round(s * 32767)));
      }

      // Base64 encode and send
      const bytes = new Uint8Array(int16.buffer);
      let binary = '';
      for (let j = 0; j < bytes.length; j++) binary += String.fromCharCode(bytes[j]);
      send('voice-data', { audio: btoa(binary), sr: TARGET_SAMPLE_RATE });
    };

    source.connect(captureProcessor);
    captureProcessor.connect(captureCtx.destination);
    send('voice-join', {});
  } catch (err) {
    console.error('[VOICE] Mic error:', err);
    if (err.name === 'NotAllowedError' || err.name === 'PermissionDeniedError') {
      alert('Microphone permission blocked. Go to browser Settings → Site Settings → Microphone and allow this site.');
    } else if (err.name === 'NotFoundError' || err.name === 'DevicesNotFoundError') {
      alert('No microphone found on this device.');
    } else if (err.name === 'NotReadableError' || err.name === 'TrackStartError') {
      alert('Microphone is in use by another app. Close other apps using the mic and try again.');
    } else {
      alert('Microphone error: ' + err.name + ' - ' + err.message);
    }
  }
}

function stopMic() {
  voiceMicOn = false;
  if (localStream) {
    localStream.getTracks().forEach(t => t.stop());
    localStream = null;
  }
  if (captureProcessor) { captureProcessor.disconnect(); captureProcessor = null; }
  if (captureCtx) { captureCtx.close(); captureCtx = null; }
  send('voice-leave', {});
}

// ---- Speaker toggle ----
function toggleSpeaker() {
  ensurePlaybackCtx(); // user gesture — ensure playback ctx is alive
  voiceSpeakerOn = !voiceSpeakerOn;
  updateVoiceUI();
}

// ---- Playback ----
function playAudioChunk(peerId, base64Audio, sampleRate) {
  if (!voiceSpeakerOn) return;

  var ctx;
  try { ctx = ensurePlaybackCtx(); } catch(e) { return; }

  // Decode base64 → Int16 → Float32
  const binary = atob(base64Audio);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  const int16 = new Int16Array(bytes.buffer);
  const float32 = new Float32Array(int16.length);
  for (let i = 0; i < int16.length; i++) float32[i] = int16[i] / 32768.0;

  // Create and schedule buffer for gapless playback
  const sr = sampleRate || TARGET_SAMPLE_RATE;
  const buffer = ctx.createBuffer(1, float32.length, sr);
  buffer.getChannelData(0).set(float32);

  const src = ctx.createBufferSource();
  src.buffer = buffer;
  src.connect(ctx.destination);

  const now = ctx.currentTime;
  if (!peerNextTime[peerId] || peerNextTime[peerId] < now) {
    peerNextTime[peerId] = now + 0.02; // tiny initial buffer
  }
  src.start(peerNextTime[peerId]);
  peerNextTime[peerId] += buffer.duration;
}

// ---- Handle messages from server ----
function handleVoiceMessage(msg) {
  const payload = msg.payload;
  switch (msg.type) {
    case 'voice-join':
      updateVoiceUI();
      break;
    case 'voice-leave':
      delete peerNextTime[payload.peerId];
      updateVoiceUI();
      break;
    case 'voice-data':
      playAudioChunk(payload.peerId, payload.audio, payload.sr);
      break;
  }
}

// ---- Cleanup ----
function voiceCleanup() {
  stopMic();
  if (playbackCtx && playbackCtx.state !== 'closed') {
    playbackCtx.close();
  }
  playbackCtx = null;
  for (const pid of Object.keys(peerNextTime)) delete peerNextTime[pid];
}

// ---- UI update ----
function updateVoiceUI() {
  document.querySelectorAll('.voice-mic-btn').forEach(btn => {
    btn.classList.toggle('active', voiceMicOn);
    btn.innerHTML = voiceMicOn ? '🎙️' : '🎤';
    btn.title = voiceMicOn ? 'Mute Mic' : 'Unmute Mic';
  });
  document.querySelectorAll('.voice-speaker-btn').forEach(btn => {
    btn.classList.toggle('active', voiceSpeakerOn);
    btn.innerHTML = voiceSpeakerOn ? '🔊' : '🔇';
    btn.title = voiceSpeakerOn ? 'Mute Speaker' : 'Unmute Speaker';
  });
}

// ---- Cleanup on disconnect ----
function voiceCleanup() {
  stopMic();
  voiceSpeakerOn = true;
  updateVoiceUI();
}
