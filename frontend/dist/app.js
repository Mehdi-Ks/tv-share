"use strict";

const api = () => window.go.main.App;
const rt = () => window.runtime;
const $ = (id) => document.getElementById(id);

let state = null;
let movies = [];
let preparing = null; // path of the movie being prepared

// ---------- helpers ----------
function toast(msg, isError = false, ms = 4200) {
  const el = document.createElement("div");
  el.className = "toast" + (isError ? " error" : "");
  el.textContent = msg;
  $("toasts").appendChild(el);
  setTimeout(() => el.remove(), ms);
}
function errText(e) {
  return (e && e.message) || String(e || "Something went wrong");
}
function gb(mb) {
  return mb >= 1024 ? (mb / 1024).toFixed(1) + " GB" : mb + " MB";
}
function el(tag, attrs = {}, ...kids) {
  const n = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "class") n.className = v;
    else if (k === "text") n.textContent = v;
    else if (k.startsWith("on")) n.addEventListener(k.slice(2), v);
    else if (v !== false && v != null) n.setAttribute(k, v === true ? "" : v);
  }
  for (const k of kids) if (k) n.append(k);
  return n;
}
const checkSvg = () => {
  const s = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  s.setAttribute("viewBox", "0 0 16 16");
  s.innerHTML = '<path d="M3.5 8.5l3 3 6-7" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>';
  return s;
};

// ---------- state ----------
async function load() {
  state = await api().GetState();
  render();
  if (state.deps.ready) loadMovies();
  api().RefreshNetwork().then((n) => { state.network = n; render(); });
}

function render() {
  const s = state;
  const ready = s.deps.ready;
  $("setup").classList.toggle("hidden", ready);
  $("main").classList.toggle("hidden", !ready);
  $("depRclone").className = "check" + (s.deps.rclone ? " done" : "");
  $("depFfmpeg").className = "check" + (s.deps.ffmpeg ? " done" : "");

  const on = s.sharing;
  $("statusPill").className = "pill " + (on ? "pill-on" : "pill-off");
  $("statusText").textContent = on ? "Sharing" : "Not sharing";
  $("shareCard").classList.toggle("on", on);
  $("shareTitle").textContent = on ? "Your TV can see your movies" : "Share with your TV";
  $("shareSub").textContent = on
    ? `Look for “${s.settings.name}” on the TV.`
    : "Your TV can't see anything yet.";
  $("shareBtn").classList.toggle("on", on);
  $("shareBtn").setAttribute("aria-pressed", on);
  $("shareBtnText").textContent = on ? "Stop sharing" : "Start sharing";

  $("folderPath").textContent = s.settings.folder;
  $("emptyFolder").textContent = s.settings.folder;
  $("nameText").textContent = s.settings.name;
  $("nameText2").textContent = s.settings.name;

  const n = s.network || {};
  const parts = [];
  if (n.name) parts.push(n.name);
  if (n.ip) parts.push(n.ip);
  $("netText").textContent = parts.length ? parts.join(" · ") : (n.ip === "" ? "Not connected" : "Checking…");
  $("netWarn").classList.toggle("hidden", n.category !== "Public");

  $("optAutoStart").checked = s.settings.autoStartShare;
  $("optLogin").checked = s.settings.launchAtLogin;
  $("optRecycle").checked = s.settings.recycleOriginal;
  $("versionText").textContent = `TV Share ${s.version}`;
}

// ---------- setup ----------
$("installBtn").addEventListener("click", async () => {
  const btn = $("installBtn");
  btn.disabled = true;
  btn.textContent = "Installing…";
  $("setupError").classList.add("hidden");
  if (!state.deps.rclone) $("depRclone").className = "check busy";
  if (!state.deps.ffmpeg) $("depFfmpeg").className = "check busy";
  try {
    const deps = await api().InstallDeps();
    state.deps = deps;
    $("setupLog").textContent = "";
    if (deps.ready) {
      toast("All set. You can start sharing.");
      await load();
    }
  } catch (e) {
    $("setupError").textContent = errText(e);
    $("setupError").classList.remove("hidden");
    state = await api().GetState();
    render();
  } finally {
    btn.disabled = false;
    btn.textContent = "Install helpers";
  }
});

// ---------- sharing ----------
$("shareBtn").addEventListener("click", async () => {
  const btn = $("shareBtn");
  btn.disabled = true;
  $("shareError").classList.add("hidden");
  try {
    if (state.sharing) {
      await api().StopShare();
    } else {
      $("shareBtnText").textContent = "Starting…";
      await api().StartShare();
    }
  } catch (e) {
    $("shareError").textContent = "Couldn't start: " + errText(e);
    $("shareError").classList.remove("hidden");
  } finally {
    btn.disabled = false;
    state = await api().GetState();
    render();
  }
});

async function restartIfSharing() {
  if (!state.sharing) return;
  await api().StopShare();
  try { await api().StartShare(); } catch (e) { toast("Couldn't restart sharing: " + errText(e), true); }
}

$("folderBtn").addEventListener("click", async () => {
  try {
    const before = state.settings.folder;
    const dir = await api().ChooseFolder();
    if (dir && dir !== before) {
      await restartIfSharing();
      state = await api().GetState();
      render();
      loadMovies();
    }
  } catch (e) { toast(errText(e), true); }
});

$("nameBtn").addEventListener("click", () => {
  $("nameInput").value = state.settings.name;
  $("nameDlg").showModal();
  $("nameInput").select();
});
$("nameDlg").addEventListener("close", async () => {
  if ($("nameDlg").returnValue !== "save") return;
  const v = $("nameInput").value.trim();
  if (!v || v === state.settings.name) return;
  try {
    await api().SetName(v);
    await restartIfSharing();
    state = await api().GetState();
    render();
  } catch (e) { toast(errText(e), true); }
});

$("netBtn").addEventListener("click", () => api().OpenNetworkSettings());
$("helpBtn").addEventListener("click", () => api().OpenURL("https://github.com/Mehdi-Ks/tv-share#when-something-doesnt-work"));
$("openFolderBtn").addEventListener("click", () => api().OpenFolder(""));
$("refreshBtn").addEventListener("click", async () => {
  api().RefreshNetwork().then((n) => { state.network = n; render(); });
  loadMovies();
});

// ---------- settings ----------
$("settingsBtn").addEventListener("click", async () => {
  const sel = $("optLang");
  if (!sel.options.length) {
    const langs = await api().Languages();
    for (const [code, name] of langs) sel.append(el("option", { value: code, text: name }));
  }
  sel.value = state.settings.subLang;
  $("settingsDlg").showModal();
});
for (const [id, key] of [["optAutoStart", "autoStartShare"], ["optLogin", "launchAtLogin"], ["optRecycle", "recycleOriginal"]]) {
  $(id).addEventListener("change", async (ev) => {
    try {
      await api().SetOption(key, ev.target.checked);
      state.settings[key] = ev.target.checked;
    } catch (e) {
      ev.target.checked = !ev.target.checked;
      toast(errText(e), true);
    }
  });
}
$("optLang").addEventListener("change", async (ev) => {
  await api().SetSubLang(ev.target.value);
  state.settings.subLang = ev.target.value;
  loadMovies();
});

// ---------- movies ----------
async function loadMovies() {
  $("moviesSub").textContent = "Looking for movies…";
  try {
    movies = await api().ListMovies();
  } catch (e) {
    movies = [];
    toast(errText(e), true);
  }
  renderMovies();
}

function renderMovies() {
  const list = $("movieList");
  list.replaceChildren();
  $("emptyState").classList.toggle("hidden", movies.length > 0);
  const needSubs = movies.filter((m) => !m.hasSubs && m.subs.length).length;
  $("moviesSub").textContent = movies.length === 0
    ? "Nothing found yet"
    : `${movies.length} movie${movies.length > 1 ? "s" : ""}` + (needSubs ? ` · ${needSubs} can get subtitles` : "");

  for (const m of movies) list.append(movieRow(m));
}

function movieRow(m) {
  const meta = el("div", { class: "movie-meta" }, el("span", { text: gb(m.sizeMB) }));
  if (m.outside) meta.append(el("span", { class: "tag", text: "Not in your shared folder: the TV won't see it", title: m.path }));

  const actions = el("div", { class: "movie-actions" });
  const row = el("li", { class: "movie", "data-path": m.path },
    el("div", { class: "movie-title", text: m.title, title: m.path }),
    actions,
    meta,
  );

  if (m.hasSubs) {
    const b = el("span", { class: "badge badge-ok" }, checkSvg(), document.createTextNode("Subtitles built in"));
    actions.append(b);
  } else if (m.subs.length) {
    const sel = el("select", { "aria-label": "Subtitle language" });
    m.subs.forEach((s, i) => sel.append(el("option", { value: s.path, text: s.label, selected: i === m.defaultSub })));
    const btn = el("button", {
      class: "btn btn-primary",
      text: "Add subtitles",
      disabled: preparing !== null,
      onclick: () => addSubtitles(m, sel.value, row),
    });
    actions.append(sel, btn);
  } else {
    actions.append(el("span", { class: "badge badge-muted", text: "No subtitle files found" }));
  }
  return row;
}

async function addSubtitles(m, subPath, row) {
  preparing = m.path;
  document.querySelectorAll(".movie-actions .btn-primary").forEach((b) => (b.disabled = true));
  const bar = el("i");
  const pct = el("span", { text: "0%" });
  const cancel = el("button", { class: "link-btn", text: "Cancel", onclick: () => api().CancelSubtitles() });
  const progress = el("div", { class: "progress" }, el("div", { class: "bar" }, bar), pct, cancel);
  row.append(progress);
  row._setProgress = (p) => { bar.style.width = p + "%"; pct.textContent = Math.round(p) + "%"; };

  try {
    const res = await api().AddSubtitles(m.path, subPath);
    toast(res.recycled
      ? `Subtitles added to “${m.title}”. The original is in the Recycle Bin.`
      : `Subtitles added to “${m.title}”.`);
  } catch (e) {
    const msg = errText(e);
    if (msg !== "cancelled") toast(msg, true, 7000);
  } finally {
    preparing = null;
    loadMovies();
  }
}

// ---------- events from the app ----------
function wireEvents() {
  rt().EventsOn("setup", (line) => { $("setupLog").textContent = line; });
  rt().EventsOn("network", (n) => { if (state) { state.network = n; render(); } });
  rt().EventsOn("share:started", async () => { state = await api().GetState(); render(); });
  rt().EventsOn("share:stopped", async (msg) => {
    state = await api().GetState();
    render();
    if (msg) toast("Sharing stopped: " + msg, true, 7000);
  });
  rt().EventsOn("prepare:progress", (d) => {
    const row = document.querySelector(`.movie[data-path="${CSS.escape(d.path)}"]`);
    if (row && row._setProgress) row._setProgress(d.percent);
  });
  rt().EventsOn("drop", (r) => {
    $("dropOverlay").classList.add("hidden");
    if (r.message) toast(r.message, true);
    if (r.added) loadMovies();
  });
}

// Drag-and-drop highlight (the drop itself is handled by the app)
let dragDepth = 0;
window.addEventListener("dragenter", (e) => { e.preventDefault(); dragDepth++; if (state && state.deps.ready) $("dropOverlay").classList.remove("hidden"); });
window.addEventListener("dragleave", () => { dragDepth = Math.max(0, dragDepth - 1); if (!dragDepth) $("dropOverlay").classList.add("hidden"); });
window.addEventListener("dragover", (e) => e.preventDefault());
window.addEventListener("drop", (e) => { e.preventDefault(); dragDepth = 0; $("dropOverlay").classList.add("hidden"); });

// Refresh when the window comes back to the front (new downloads show up)
let lastFocusLoad = 0;
window.addEventListener("focus", () => {
  if (!state || !state.deps.ready || preparing || Date.now() - lastFocusLoad < 4000) return;
  lastFocusLoad = Date.now();
  loadMovies();
});

window.addEventListener("DOMContentLoaded", () => {
  wireEvents();
  load().catch((e) => toast(errText(e), true));
});
