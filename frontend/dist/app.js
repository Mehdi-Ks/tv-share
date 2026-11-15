"use strict";

const api = () => window.go.main.App;
const rt = () => window.runtime;
const $ = (id) => document.getElementById(id);
const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

let state = null;
let movies = [];
let preparing = null;          // path of the movie being prepared
const posters = new Map();     // path -> data URL, or "" when no frame could be taken

// ---------- helpers ----------
function toast(msg, isError = false, ms = 4200) {
  const t = el("div", { class: "toast" + (isError ? " error" : ""), text: msg });
  $("toasts").append(t);
  setTimeout(() => {
    t.classList.add("leaving");
    setTimeout(() => t.remove(), 180);
  }, ms);
}
const errText = (e) => (e && e.message) || String(e || "Something went wrong");
const gb = (mb) => (mb >= 1024 ? (mb / 1024).toFixed(1) + " GB" : mb + " MB");
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
function icon(id, cls) {
  const s = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  s.setAttribute("aria-hidden", "true");
  if (cls) s.setAttribute("class", cls);
  const u = document.createElementNS("http://www.w3.org/2000/svg", "use");
  u.setAttribute("href", "#" + id);
  s.append(u);
  return s;
}
// "Some Movie (2026)" -> { name: "Some Movie", year: "2026" }
function splitTitle(t) {
  const m = /^(.*?)\s*\((\d{4})\)$/.exec(t);
  return m ? { name: m[1], year: m[2] } : { name: t, year: "" };
}
// A warm, stable colour per title for covers without a frame
function coverColors(t) {
  let h = 0;
  for (const c of t) h = (h * 31 + c.charCodeAt(0)) >>> 0;
  const hue = [8, 22, 32, 345, 200, 265, 160][h % 7];
  return `linear-gradient(160deg, hsl(${hue} 55% 34%) 0%, hsl(${(hue + 25) % 360} 45% 16%) 70%, #0d0d0d 100%)`;
}

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
  $("searchWrap").style.visibility = ready ? "" : "hidden";
  $("shareBtn").style.visibility = ready ? "" : "hidden";
  $("depRclone").className = "check" + (s.deps.rclone ? " done" : "");
  $("depFfmpeg").className = "check" + (s.deps.ffmpeg ? " done" : "");

  const on = s.sharing;
  $("shareBtn").setAttribute("aria-checked", on);
  $("shareSub").textContent = on ? `On · TV sees “${s.settings.name}”` : "Off";
  $("shareBtn").title = on ? "Stop sharing" : "Let the TV see your movies";

  document.querySelectorAll(".tv-name").forEach((n) => (n.textContent = s.settings.name));
  $("folderPath").textContent = s.settings.folder;
  $("emptyFolder").textContent = s.settings.folder;

  const n = s.network || {};
  const parts = [n.name, n.ip].filter(Boolean);
  $("netText").textContent = parts.length ? parts.join(" · ") : "Checking…";
  $("netWarn").classList.toggle("hidden", !ready || n.category !== "Public");

  $("optAutoStart").checked = s.settings.autoStartShare;
  $("optLogin").checked = s.settings.launchAtLogin;
  $("optRecycle").checked = s.settings.recycleOriginal;
  $("versionText").textContent = `TV Share ${s.version} · free and open source`;
}

function showShareError(msg) {
  $("shareError").textContent = msg;
  $("shareError").classList.toggle("hidden", !msg);
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
      toast("All set. Turn on “Share with TV” to start.");
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
  showShareError("");
  try {
    if (state.sharing) {
      await api().StopShare();
    } else {
      $("shareSub").textContent = "Starting…";
      await api().StartShare();
      if (!localStorageGet("seenHelp")) { openDrawer(); localStorageSet("seenHelp", "1"); }
    }
  } catch (e) {
    showShareError("Couldn't start sharing: " + errText(e));
  } finally {
    btn.disabled = false;
    state = await api().GetState();
    render();
  }
});

function localStorageGet(k) { try { return localStorage.getItem(k); } catch { return null; } }
function localStorageSet(k, v) { try { localStorage.setItem(k, v); } catch {} }

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
$("openFolderBtn").addEventListener("click", () => api().OpenFolder(""));
$("emptyOpenBtn").addEventListener("click", () => api().OpenFolder(""));
$("refreshBtn").addEventListener("click", async () => {
  api().RefreshNetwork().then((n) => { state.network = n; render(); });
  loadMovies();
});
$("helpLink").addEventListener("click", () => api().OpenURL("https://github.com/Mehdi-Ks/tv-share#when-something-doesnt-work"));

// ---------- drawer ----------
let drawerReturn = null;
function openDrawer() {
  drawerReturn = document.activeElement;
  $("scrim").hidden = false;
  requestAnimationFrame(() => $("scrim").classList.add("open"));
  $("drawer").classList.add("open");
  $("drawer").setAttribute("aria-hidden", "false");
  setTimeout(() => $("drawerClose").focus(), 50);
}
function closeDrawer() {
  $("scrim").classList.remove("open");
  $("drawer").classList.remove("open");
  $("drawer").setAttribute("aria-hidden", "true");
  setTimeout(() => { $("scrim").hidden = true; }, 220);
  if (drawerReturn) drawerReturn.focus();
}
$("helpBtn").addEventListener("click", openDrawer);
$("drawerClose").addEventListener("click", closeDrawer);
$("scrim").addEventListener("click", closeDrawer);
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && $("drawer").classList.contains("open")) closeDrawer();
  if ((e.ctrlKey && e.key.toLowerCase() === "f") || (e.key === "/" && document.activeElement.tagName !== "INPUT")) {
    e.preventDefault();
    $("search").focus();
  }
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
let firstRender = true;
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

$("search").addEventListener("input", () => applySearch());
function applySearch() {
  const q = $("search").value.trim().toLowerCase();
  let shown = 0;
  document.querySelectorAll("#grid .card").forEach((c) => {
    const hit = !q || c.dataset.title.includes(q);
    c.classList.toggle("hidden", !hit);
    if (hit) shown++;
  });
  $("noMatch").classList.toggle("hidden", !(movies.length && q && shown === 0));
}

function renderMovies() {
  const grid = $("grid");
  grid.replaceChildren();
  $("emptyState").classList.toggle("hidden", movies.length > 0);
  const canAdd = movies.filter((m) => !m.hasSubs && m.subs.length).length;
  $("moviesSub").textContent = movies.length === 0
    ? "Nothing found yet"
    : `${movies.length} movie${movies.length > 1 ? "s" : ""}` + (canAdd ? ` · ${canAdd} can get subtitles` : "");

  movies.forEach((m, i) => {
    const card = movieCard(m);
    if (firstRender && !reduceMotion && i < 24) {
      card.classList.add("card-in");
      card.style.animationDelay = i * 35 + "ms";
    }
    grid.append(card);
  });
  firstRender = false;
  applySearch();
  observePosters();
}

function movieCard(m) {
  const { name, year } = splitTitle(m.title);
  const poster = el("div", { class: "poster" });

  // Cover (always there; the frame fades in on top when ready)
  const cover = el("div", { class: "cover" }, icon("i-tv", "cover-tv"), document.createTextNode(name), year ? el("small", { text: year }) : null);
  cover.style.background = coverColors(m.title);
  poster.append(cover);
  const img = el("img", { alt: "", decoding: "async" });
  img.addEventListener("load", () => img.classList.add("loaded"));
  poster.append(img);
  const cached = posters.get(m.path);
  if (cached) img.src = cached;

  if (m.hasSubs) poster.append(el("span", { class: "corner corner-ok", title: "Subtitles built in" }, icon("i-cc"), document.createTextNode("CC")));
  else if (m.subs.length) poster.append(el("span", { class: "corner", title: "Subtitles can be added" }, document.createTextNode("SRT")));

  // Hover / focus panel
  const overlay = el("div", { class: "overlay" });
  if (m.hasSubs) {
    overlay.append(el("div", { class: "done" }, icon("i-check"), document.createTextNode("Subtitles built in")));
  } else if (m.subs.length) {
    const sel = el("select", { "aria-label": `Subtitle language for ${name}` });
    m.subs.forEach((s, i) => sel.append(el("option", { value: s.path, text: s.label, selected: i === m.defaultSub })));
    overlay.append(sel, el("button", {
      class: "btn btn-primary", disabled: preparing !== null,
      onclick: (e) => { e.stopPropagation(); addSubtitles(m, sel.value, poster); },
    }, icon("i-plus"), document.createTextNode("Add subtitles")));
  } else {
    overlay.append(el("div", { class: "muted small", text: "No subtitle files found" }));
  }
  overlay.append(el("button", { class: "link-btn file-link", text: "Show in folder", onclick: () => api().ShowInFolder(m.path) }));
  poster.append(overlay);

  // Under the poster
  const meta = el("div", { class: "card-meta" });
  if (year) meta.append(el("span", { text: year }));
  meta.append(el("span", { text: gb(m.sizeMB) }));
  if (m.outside) meta.append(el("span", { class: "tag", text: "Not shared", title: "Outside your shared folder: the TV won't see it" }));
  if (m.hasSubs) meta.append(el("span", { class: "ok" }, icon("i-cc"), document.createTextNode("CC")));
  else if (m.subs.length) meta.append(el("button", {
    class: "add", disabled: preparing !== null, title: `Add ${m.subs[m.defaultSub].label} subtitles`,
    onclick: () => addSubtitles(m, m.subs[m.defaultSub].path, poster),
  }, icon("i-plus"), document.createTextNode("Subtitles")));

  const statusText = m.hasSubs ? "subtitles built in" : m.subs.length ? "subtitles can be added" : "no subtitles";
  return el("li", {
    class: "card", tabindex: "0", "data-path": m.path, "data-title": m.title.toLowerCase(),
    "aria-label": `${m.title}, ${gb(m.sizeMB)}, ${statusText}`,
  }, poster, el("div", { class: "card-title", text: name, title: m.path }), meta);
}

// Posters load only when their card scrolls into view.
const posterObserver = new IntersectionObserver((entries) => {
  for (const en of entries) {
    if (!en.isIntersecting) continue;
    posterObserver.unobserve(en.target);
    const path = en.target.dataset.path;
    if (posters.has(path)) continue;
    api().GetPoster(path).then((url) => {
      posters.set(path, url);
      const img = en.target.querySelector(".poster img");
      if (img) img.src = url;
    }).catch(() => posters.set(path, ""));
  }
}, { rootMargin: "300px" });
function observePosters() {
  document.querySelectorAll("#grid .card").forEach((c) => {
    if (!posters.has(c.dataset.path)) posterObserver.observe(c);
  });
}

async function addSubtitles(m, subPath, poster) {
  preparing = m.path;
  document.querySelectorAll(".overlay .btn-primary, .card-meta .add").forEach((b) => (b.disabled = true));
  const label = el("span", { text: "0%" });
  const ring = el("div", { class: "ring" }, label);
  const busy = el("div", { class: "busy" }, ring, el("small", { text: "Adding subtitles…" }),
    el("button", { class: "link-btn", text: "Cancel", onclick: () => api().CancelSubtitles() }));
  poster.append(busy);
  poster.closest(".card").dataset.busy = "1";

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
    if (msg) showShareError("Sharing stopped: " + msg);
  });
  rt().EventsOn("prepare:progress", (d) => {
    const card = document.querySelector(`.card[data-path="${CSS.escape(d.path)}"]`);
    if (!card) return;
    const ring = card.querySelector(".ring");
    if (ring) { ring.style.setProperty("--p", d.percent.toFixed(1)); ring.querySelector("span").textContent = Math.round(d.percent) + "%"; }
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

// New downloads show up when the window comes back to the front
let lastFocusLoad = Date.now();
window.addEventListener("focus", () => {
  if (!state || !state.deps.ready || preparing || Date.now() - lastFocusLoad < 4000) return;
  lastFocusLoad = Date.now();
  loadMovies();
});

window.addEventListener("DOMContentLoaded", () => {
  wireEvents();
  load().catch((e) => toast(errText(e), true));
});
