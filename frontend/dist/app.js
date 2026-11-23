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
// toast(msg, isError, ms, extra...): extra nodes (a hint, an action button) go after the message.
function toast(msg, isError = false, ms = 4200, ...extra) {
  const t = el("div", { class: "toast" + (isError ? " error" : "") + (extra.length ? " has-action" : "") }, el("span", { text: msg }), ...extra);
  const box = $("toasts");
  // Messages live in the top layer, above the movie page and dialogs (which are in it too), and
  // inside the open one: everything outside an open dialog can't be clicked.
  const host = [...document.querySelectorAll("dialog[open]")].pop() || document.body;
  if (box.parentElement !== host) host.append(box);
  box.append(t);
  if (box.showPopover) {
    try { if (box.matches(":popover-open")) box.hidePopover(); box.showPopover(); } catch {}
  }
  // A message with a button stays while the pointer is on it, so it can't vanish mid-click.
  const leave = () => {
    if (t.matches(":hover")) { setTimeout(leave, 1500); return; }
    t.classList.add("leaving");
    setTimeout(() => {
      t.remove();
      if (!box.children.length && box.hidePopover) { try { box.hidePopover(); } catch {} }
    }, 180);
  };
  setTimeout(leave, extra.length ? Math.max(ms, 20000) : ms);
}
// When a dialog holding the messages closes, move them to what's still open so they don't vanish.
document.addEventListener("close", (e) => {
  const box = $("toasts");
  if (e.target.tagName !== "DIALOG" || !e.target.contains(box)) return;
  (([...document.querySelectorAll("dialog[open]")].pop()) || document.body).append(box);
  if (box.children.length && box.showPopover) { try { box.showPopover(); } catch {} }
}, true);
function localStorageGet(k) { try { return localStorage.getItem(k); } catch { return null; } }
function localStorageSet(k, v) { try { localStorage.setItem(k, v); } catch {} }
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
  if (window.i18n) { try { await api().SetLanguage(window.i18n.lang); } catch {} }
  try { window.__langs = await api().Languages(); } catch {}
  state = await api().GetState();
  render();
  if (state.deps.ready && state.settings.setupDone) loadMovies();
  api().RefreshNetwork().then((n) => { state.network = n; render(); });
}

function render() {
  const s = state;
  const ready = s.deps.ready && s.settings.setupDone;
  $("setup").classList.toggle("hidden", s.deps.ready);
  const welcome = s.deps.ready && !s.settings.setupDone;
  if (welcome && $("welcome").classList.contains("hidden")) showWelcome();
  $("welcome").classList.toggle("hidden", !welcome);
  $("main").classList.toggle("hidden", !ready);
  $("searchWrap").style.visibility = ready ? "" : "hidden";
  $("shareBtn").style.visibility = ready ? "" : "hidden";
  $("settingsBtn").style.visibility = ready ? "" : "hidden";
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

  const t = s.tmdb || {};
  $("tmdbAsk").classList.toggle("hidden", !(ready && t.available && !s.settings.tmdb));
  $("optTMDB").checked = s.settings.tmdb === "on";
  $("optTMDB").disabled = !t.available;
  $("keyNote").textContent = t.ownKey
    ? "Using your own key. Empty the box and press Save to go back to the app's key."
    : t.builtIn
      ? "Optional. This app already has one; use yours if you prefer."
      : "Needed for official posters. Get a free key at themoviedb.org (Settings, API).";
  $("keyInput").placeholder = t.ownKey ? "Your key is saved" : "Paste a key";
  $("optAutoStart").checked = s.settings.autoStartShare;
  $("optLogin").checked = s.settings.launchAtLogin;
  $("optRecycle").checked = s.settings.recycleOriginal;
  $("optSync").checked = !s.settings.noAutoSync;
  $("versionText").textContent = `TV Share ${s.version} · free and open source`;
}

function showShareError(msg) {
  $("shareError").textContent = msg;
  $("shareError").classList.toggle("hidden", !msg);
}

// ---------- setup ----------
$("storeBtn").addEventListener("click", () => api().OpenAppInstallerStore());
$("installBtn").addEventListener("click", async () => {
  const btn = $("installBtn");
  btn.disabled = true;
  btn.textContent = "Installing…";
  $("setupError").classList.add("hidden");
  if (!state.deps.rclone) $("depRclone").className = "check spinning";
  if (!state.deps.ffmpeg) $("depFfmpeg").className = "check spinning";
  try {
    const deps = await api().InstallDeps();
    state.deps = deps;
    $("setupLog").textContent = "";
    if (deps.ready) {
      await load();
    }
  } catch (e) {
    $("setupError").textContent = errText(e);
    $("setupError").classList.remove("hidden");
    $("storeBtn").classList.remove("hidden");
    state = await api().GetState();
    render();
  } finally {
    btn.disabled = false;
    btn.textContent = "Install helpers";
  }
});

// ---------- first run: folder and name ----------
let wFolder = "";
async function showWelcome() {
  wFolder = state.settings.folder;
  $("wName").value = state.settings.name;
  $("wPosters").classList.toggle("hidden", !(state.tmdb && state.tmdb.available));
  await updateWelcomeFolder();
}
async function updateWelcomeFolder() {
  $("wFolder").textContent = wFolder;
  $("wCount").className = "muted";
  $("wCount").textContent = "Looking for movies…";
  const n = await api().CountMovies(wFolder);
  $("wCount").textContent = n < 0 ? "This folder doesn't exist. Choose another one."
    : n === 0 ? "No movies here yet. That's fine, you can add them later."
    : `${n} movie${n > 1 ? "s" : ""} found`;
  if (n > 0) $("wCount").className = "found";
}
$("wChoose").addEventListener("click", async () => {
  try {
    const dir = await api().PickFolder(wFolder);
    if (dir) { wFolder = dir; await updateWelcomeFolder(); }
  } catch (e) { toast(errText(e), true); }
});
$("welcomeForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  $("wError").classList.add("hidden");
  $("wDone").disabled = true;
  try {
    const posterChoice = document.querySelector('input[name="wPoster"]:checked');
    if (!$("wPosters").classList.contains("hidden")) {
      if (!posterChoice) throw new Error("Choose how your posters should look.");
      await api().SetTMDBChoice(posterChoice.value === "on");
    }
    await api().FinishSetup(wFolder, $("wName").value);
    await load();
    toast("All set. Turn on Share with TV to show your movies on the TV.");
  } catch (err) {
    $("wError").textContent = errText(err);
    $("wError").classList.remove("hidden");
  } finally {
    $("wDone").disabled = false;
  }
});

// ---------- sharing ----------
// The first share makes Windows Firewall ask about "Rclone", a name people don't know.
// Explain it once, just before it happens.
function firewallHeadsUp() {
  if (localStorageGet("seenFirewall")) return Promise.resolve(true);
  return new Promise((resolve) => {
    const dlg = $("fwDlg");
    dlg.addEventListener("close", () => {
      const go = dlg.returnValue === "go";
      if (go) localStorageSet("seenFirewall", "1");
      resolve(go);
    }, { once: true });
    dlg.showModal();
  });
}

$("shareBtn").addEventListener("click", async () => {
  if (!state.sharing && !(await firewallHeadsUp())) return;
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

// Clicking the dimmed area outside a dialog closes it, like Esc.
for (const d of document.querySelectorAll("dialog.dialog")) {
  d.addEventListener("click", (e) => {
    if (e.target !== d) return;
    const r = d.getBoundingClientRect();
    const inside = e.clientX >= r.left && e.clientX <= r.right && e.clientY >= r.top && e.clientY <= r.bottom;
    if (!inside) d.close("cancel");
  });
}

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
  setBackgroundInert(true);
  setTimeout(() => $("drawerClose").focus(), 50);
}
function closeDrawer() {
  $("scrim").classList.remove("open");
  $("drawer").classList.remove("open");
  $("drawer").setAttribute("aria-hidden", "true");
  setBackgroundInert(false);
  setTimeout(() => { $("scrim").hidden = true; }, 220);
  if (drawerReturn) drawerReturn.focus();
}
// While the panel is open, keyboard and screen readers stay inside it.
function setBackgroundInert(on) {
  for (const n of [$("main"), $("setup"), $("welcome"), document.querySelector(".topbar")]) n.inert = on;
}
$("helpBtn").addEventListener("click", openDrawer);
// Delegated: the translated help text replaces this link's element.
document.addEventListener("click", (e) => {
  if (e.target.closest && e.target.closest("#drawerSettings")) { closeDrawer(); $("settingsBtn").click(); }
});
$("drawerClose").addEventListener("click", closeDrawer);
$("scrim").addEventListener("click", closeDrawer);
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && $("drawer").classList.contains("open")) closeDrawer();
  if ($("detail").open || $("showPage").open) return;
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
  refreshOSStatus();
  $("settingsDlg").showModal();
});
for (const [id, key] of [["optAutoStart", "autoStartShare"], ["optLogin", "launchAtLogin"], ["optRecycle", "recycleOriginal"], ["optSync", "autoSync"]]) {
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
// ---------- official posters (TMDB) ----------
const tmdbOn = () => state && state.settings.tmdb === "on" && state.tmdb && state.tmdb.available;
const metaCache = new Map();

async function setTMDB(on) {
  try {
    await api().SetTMDBChoice(on);
    state = await api().GetState();
    render();
    resetArt();
  } catch (e) { toast(errText(e), true); }
}
$("tmdbYes").addEventListener("click", () => setTMDB(true));
$("tmdbNo").addEventListener("click", () => setTMDB(false));
$("optTMDB").addEventListener("change", (e) => setTMDB(e.target.checked));
$("keySave").addEventListener("click", async () => {
  const btn = $("keySave");
  btn.disabled = true;
  try {
    const key = $("keyInput").value.trim();
    await api().SetTMDBKey(key);
    $("keyInput").value = "";
    toast(key ? "Key saved. TMDB accepted it." : "Back to the app's own key.");
    state = await api().GetState();
    render();
    resetArt();
  } catch (e) { toast(errText(e), true); }
  finally { btn.disabled = false; }
});

// Posters, backdrops and info all change when TMDB is turned on or off: start over.
function resetArt() {
  posters.clear(); backdrops.clear(); metaCache.clear(); showMetaCache.clear(); seasonCache.clear(); stills.clear();
  renderMovies();
}
// One movie's match changed: refresh just that one.
function resetOne(path) {
  posters.delete(path); backdrops.delete(path); metaCache.delete(path);
  fetchPoster(path).then((url) => {
    posters.set(path, url);
    document.querySelectorAll(`[data-path="${CSS.escape(path)}"] .poster img, .detail-poster[data-path="${CSS.escape(path)}"] img`)
      .forEach((img) => { img.classList.remove("loaded"); img.src = url; });
  }).catch(() => posters.set(path, ""));
  if (detailMovie && detailMovie.path === path) fillDetail(detailMovie, true);
}

// ---------- "Not this movie?" ----------
let matchFor = null;
async function openMatch(m) {
  matchFor = m;
  const { name, year } = splitTitle(m.title);
  $("matchQuery").value = year ? `${name} (${year})` : name;
  $("matchTitle").textContent = m.kind === "show" ? "Which show is it?" : "Which movie is it?";
  $("matchFrame").textContent = m.kind === "show" ? "Use a frame from an episode instead" : "Use a frame from the movie instead";
  $("matchDlg").showModal();
  await runMatchSearch();
}
async function runMatchSearch() {
  const list = $("matchList");
  list.replaceChildren(el("li", { class: "muted small", text: "Searching TMDB…" }));
  try {
    const found = matchFor.kind === "show"
      ? await api().FindTMDBShowMatches(matchFor.title, $("matchQuery").value)
      : await api().FindTMDBMatches(matchFor.path, $("matchQuery").value);
    const cur = ((matchFor.kind === "show" ? showMetaCache.get(matchFor.key) : metaCache.get(matchFor.path)) || {}).id;
    list.replaceChildren();
    if (!found.length) list.append(el("li", { class: "muted small", text: "Nothing found. Try another spelling, or leave out the year." }));
    for (const c of found) {
      const thumb = c.poster ? el("img", { src: c.poster, alt: "" }) : el("span", { class: "thumb" });
      list.append(el("li", {}, el("button", {
        type: "button", class: c.id === cur ? "current" : "",
        onclick: () => chooseMatch(c.id),
      }, thumb, el("span", {}, el("strong", { text: c.title, "data-no-tr": true }), el("small", { text: c.year || "Year unknown" })))));
    }
  } catch (e) {
    list.replaceChildren(el("li", { class: "error", text: errText(e) }));
  }
}
async function chooseMatch(id) {
  const m = matchFor;
  $("matchDlg").close();
  try {
    if (m.kind === "show") { await api().SetTMDBShowMatch(m.title, id); resetShow(m); }
    else { await api().SetTMDBMatch(m.path, id); resetOne(m.path); }
  } catch (e) { toast(errText(e), true); }
}
$("matchGo").addEventListener("click", runMatchSearch);
$("matchQuery").addEventListener("keydown", (e) => { if (e.key === "Enter") { e.preventDefault(); runMatchSearch(); } });
$("matchFrame").addEventListener("click", () => chooseMatch(-1));

$("optUiLang").value = window.i18n ? window.i18n.choice : "auto";
$("optUiLang").addEventListener("change", (ev) => window.i18n && window.i18n.set(ev.target.value));

$("optLang").addEventListener("change", async (ev) => {
  await api().SetSubLang(ev.target.value);
  state.settings.subLang = ev.target.value;
  loadMovies();
});

// ---------- movies ----------
let filter = "all";
let sortBy = localStorageGet("sort") || "added";
$("sortSel").value = sortBy;

async function loadMovies() {
  $("moviesSub").textContent = "Looking for movies…";
  try {
    // A movie with no subtitle files at all must still get a list, never null.
    movies = ((await api().ListMovies()) || []).map((m) => ({ ...m, subs: m.subs || [] }));
  } catch (e) {
    movies = [];
    toast(errText(e), true);
  }
  renderMovies();
}

const canGetSubs = (m) => !m.hasSubs && m.subs.length > 0;
const entryCanSubs = (x) => (x.kind === "show" ? x.canSubs > 0 : canGetSubs(x));

// Episodes become one entry per show; extras go to their show's or movie's page.
let shows = new Map();     // show key -> { kind: "show", key, id, title, episodes, extras, seasons, ... }
let extrasFor = new Map(); // movie path or "show:<key>" -> extras
let homeItems = [];        // what the home screen shows: movies and shows
function organizeItems() {
  shows = new Map(); extrasFor = new Map();
  const list = [];
  for (const m of movies) {
    if (m.kind === "episode") {
      let sh = shows.get(m.showKey);
      if (!sh) {
        sh = { kind: "show", key: m.showKey, id: "show:" + m.showKey, title: m.show, episodes: [], extras: [], sizeMB: 0, added: 0 };
        shows.set(m.showKey, sh);
        list.push(sh);
      }
      sh.episodes.push(m);
      sh.sizeMB += m.sizeMB;
      sh.added = Math.max(sh.added, m.added);
    } else if (m.kind === "extra" && m.extraFor) {
      if (!extrasFor.has(m.extraFor)) extrasFor.set(m.extraFor, []);
      extrasFor.get(m.extraFor).push(m);
    } else list.push(m);
  }
  for (const sh of shows.values()) {
    sh.episodes.sort((a, b) => a.season - b.season || a.episode - b.episode);
    sh.extras = extrasFor.get(sh.id) || [];
    // Season 0 is "Specials": listed after the real seasons.
    sh.seasons = [...new Set(sh.episodes.map((e) => e.season))].sort((a, b) => (a || 1e9) - (b || 1e9));
    sh.hasSubs = sh.episodes.every((e) => e.hasSubs);
    sh.canSubs = sh.episodes.filter(canGetSubs).length;
  }
  homeItems = list;
}
// "WandaVision S01E03" in messages; movies keep their title.
function itemName(m) {
  if (m.kind !== "episode") return m.title;
  return `${splitTitle(m.show).name} S${String(m.season).padStart(2, "0")}E${String(m.episode).padStart(2, "0")}`;
}

$("search").addEventListener("input", () => applyView());
$("sortSel").addEventListener("change", (e) => { sortBy = e.target.value; localStorageSet("sort", sortBy); renderMovies(); });
document.querySelectorAll(".chip[data-filter]").forEach((c) => c.addEventListener("click", () => {
  filter = c.dataset.filter;
  document.querySelectorAll(".chip[data-filter]").forEach((x) => x.setAttribute("aria-pressed", x === c));
  applyView();
}));

// Search and the filter chips hide cards; the grid itself stays in place.
function applyView() {
  const q = $("search").value.trim().toLowerCase();
  let shown = 0;
  document.querySelectorAll("#grid .card").forEach((c) => {
    const f = filter;
    const hit = (!q || c.dataset.title.includes(q)) && (f === "all" || (f === "subs" && c.dataset.subs === "1")
      || (f === "movies" && c.dataset.kind === "movie") || (f === "shows" && c.dataset.kind === "show"));
    c.classList.toggle("hidden", !hit);
    if (hit) shown++;
  });
  $("noMatch").textContent = q ? "No movie matches your search." : "Every movie already has its subtitles.";
  $("noMatch").classList.toggle("hidden", !(homeItems.length && shown === 0));
  fixRovingFocus();
}

function sortedEntries() {
  const list = homeItems.slice();
  if (sortBy === "added") list.sort((a, b) => b.added - a.added);
  else if (sortBy === "size") list.sort((a, b) => b.sizeMB - a.sizeMB);
  else list.sort((a, b) => a.title.localeCompare(b.title, undefined, { sensitivity: "base" }));
  return list;
}

function renderMovies() {
  organizeItems();
  const grid = $("grid");
  grid.replaceChildren();
  const n = homeItems.length;
  $("emptyState").classList.toggle("hidden", n > 0);
  $("shelfTools").classList.toggle("hidden", n === 0);
  const nShows = shows.size, nMovies = n - nShows;
  const canAdd = homeItems.filter(entryCanSubs).length;
  $("countAll").textContent = n;
  $("countSubs").textContent = canAdd;
  $("countMovies").textContent = nMovies;
  $("countShows").textContent = nShows;
  const both = nShows > 0 && nMovies > 0;
  $("kindChips").classList.toggle("hidden", !both);
  const counts = [];
  if (nMovies || !nShows) counts.push(`${nMovies} movie${nMovies === 1 ? "" : "s"}`);
  if (nShows) counts.push(`${nShows} show${nShows === 1 ? "" : "s"}`);
  $("moviesSub").textContent = n === 0 ? "Nothing found yet" : counts.join(", ") + (canAdd ? `, ${canAdd} can get subtitles` : "");
  if ((!canAdd && filter === "subs") || (!both && (filter === "movies" || filter === "shows"))) document.querySelector('.chip[data-filter="all"]').click();

  for (const x of sortedEntries()) grid.append(x.kind === "show" ? showCard(x) : movieCard(x));
  applyView();
  observePosters();
  if ($("showPage").open && showOpen) {
    const fresh = shows.get(showOpen.key);
    if (fresh) fillShow(fresh); else closeShow();
  }
  if (detailMovie) {
    const fresh = movies.find((m) => m.path === detailMovie.path) || movies.find((m) => m.title === detailMovie.title && m.kind === detailMovie.kind && m.episode === detailMovie.episode);
    if (fresh) fillDetail(fresh); else closeDetail();
  }
}

const posterKey = (x) => x.id || x.path;
function fetchPoster(key) {
  if (key.startsWith("show:")) {
    const sh = shows.get(key.slice(5));
    return sh ? api().GetShowPoster(sh.title, sh.episodes[0].path) : Promise.reject(new Error("gone"));
  }
  return api().GetPoster(key);
}
function posterEl(m) {
  const { name, year } = splitTitle(m.title);
  const poster = el("div", { class: "poster" });
  const cover = el("div", { class: "cover", "data-no-tr": true }, icon("i-tv", "cover-tv"), document.createTextNode(name), year ? el("small", { text: year }) : null);
  cover.style.background = coverColors(m.title);
  poster.append(cover);
  const img = el("img", { alt: "", decoding: "async" });
  img.addEventListener("load", () => img.classList.add("loaded"));
  poster.append(img);
  const cached = posters.get(posterKey(m)) || (m.kind === "episode" && posters.get("show:" + m.showKey));
  if (cached) img.src = cached;
  const canAdd = m.kind === "show" ? m.canSubs > 0 : m.subs.length > 0;
  if (m.hasSubs) poster.append(el("span", { class: "corner corner-ok", title: "Subtitles built in" }, icon("i-cc"), document.createTextNode("CC")));
  else if (canAdd) poster.append(el("span", { class: "corner", title: "Subtitles can be added" }, document.createTextNode("SRT")));
  return poster;
}

function movieCard(m) {
  const { name, year } = splitTitle(m.title);
  const poster = posterEl(m);

  const meta = el("div", { class: "card-meta" });
  if (year) meta.append(el("span", { text: year }));
  meta.append(el("span", { text: gb(m.sizeMB) }));
  if (m.outside) meta.append(el("span", { class: "tag", text: "Not shared", title: "Outside your shared folder: the TV won't see it" }));
  if (m.hasSubs) meta.append(el("span", { class: "ok" }, icon("i-cc"), document.createTextNode("CC")));
  else if (m.subs.length) meta.append(el("button", {
    class: "add", tabindex: "-1", disabled: preparing !== null, title: `Add ${m.subs[m.defaultSub].label} subtitles`,
    onclick: (e) => { e.stopPropagation(); addSubtitles(m, m.subs[m.defaultSub].path); },
  }, icon("i-plus"), document.createTextNode("Subtitles")));

  const statusText = m.hasSubs ? "subtitles built in" : m.subs.length ? "subtitles can be added" : "no subtitles";
  const card = el("li", {
    class: "card", tabindex: "-1", "data-path": m.path, "data-kind": "movie", "data-title": m.title.toLowerCase(), "data-subs": canGetSubs(m) ? "1" : "0",
    "aria-label": `${m.title}, ${gb(m.sizeMB)}, ${statusText}. Press Enter for details.`,
  }, poster, el("div", { class: "card-title", text: name, title: m.path, "data-no-tr": true }), meta);
  card.addEventListener("click", () => { cardClickAt = Date.now(); openDetail(m, card); });
  card.addEventListener("dblclick", () => playOnPC(m)); // the page opened on the first click stays behind the player
  card.addEventListener("keydown", (e) => {
    if (e.target === card && (e.key === "Enter" || e.key === " ")) { e.preventDefault(); openDetail(m, card); }
  });
  if (preparing === m.path) showBusy(poster);
  return card;
}

function showCard(sh) {
  const { name, year } = splitTitle(sh.title);
  const poster = posterEl(sh);
  const meta = el("div", { class: "card-meta" });
  if (year) meta.append(el("span", { text: year }));
  const nEp = uniqueEpisodes(sh);
  const nSeasons = sh.seasons.filter((n) => n > 0).length;
  meta.append(el("span", { text: nSeasons > 1 ? `${nSeasons} seasons` : `${nEp} episode${nEp === 1 ? "" : "s"}` }));
  if (sh.hasSubs) meta.append(el("span", { class: "ok" }, icon("i-cc"), document.createTextNode("CC")));
  const words = [sh.title, ...sh.episodes.map((e) => e.epTitle || "")].join(" ").toLowerCase();
  const card = el("li", {
    class: "card", tabindex: "-1", "data-path": sh.id, "data-kind": "show", "data-title": words, "data-subs": sh.canSubs ? "1" : "0",
    "aria-label": `${sh.title}, ${nEp} episodes. Press Enter for details.`,
  }, poster, el("div", { class: "card-title", text: name, title: sh.title, "data-no-tr": true }), meta);
  card.addEventListener("click", () => openShow(sh, card));
  card.addEventListener("keydown", (e) => {
    if (e.target === card && (e.key === "Enter" || e.key === " ")) { e.preventDefault(); openShow(sh, card); }
  });
  return card;
}

// ---------- keyboard: arrows move across the shelf (one tab stop for the whole grid) ----------
const visibleCards = () => [...document.querySelectorAll("#grid .card:not(.hidden)")];
function fixRovingFocus() {
  const cards = visibleCards();
  if (!cards.length) return;
  const cur = cards.find((c) => c.tabIndex === 0);
  document.querySelectorAll("#grid .card").forEach((c) => (c.tabIndex = -1));
  (cur || cards[0]).tabIndex = 0;
}
function moveFocus(to) {
  const cards = visibleCards();
  if (!cards[to]) return;
  cards.forEach((c) => (c.tabIndex = -1));
  cards[to].tabIndex = 0;
  cards[to].focus();
  cards[to].scrollIntoView({ block: "nearest" });
}
$("grid").addEventListener("keydown", (e) => {
  const cards = visibleCards();
  const i = cards.indexOf(document.activeElement);
  if (i < 0) return;
  const cols = Math.max(1, cards.filter((c) => c.offsetTop === cards[0].offsetTop).length);
  const step = { ArrowRight: 1, ArrowLeft: -1, ArrowDown: cols, ArrowUp: -cols }[e.key];
  if (step !== undefined) { e.preventDefault(); moveFocus(Math.min(cards.length - 1, Math.max(0, i + step))); }
  else if (e.key === "Home") { e.preventDefault(); moveFocus(0); }
  else if (e.key === "End") { e.preventDefault(); moveFocus(cards.length - 1); }
});
$("grid").addEventListener("focusin", (e) => {
  if (!e.target.classList.contains("card")) return;
  document.querySelectorAll("#grid .card").forEach((c) => (c.tabIndex = c === e.target ? 0 : -1));
});

// Posters load only when their card scrolls into view.
const posterObserver = new IntersectionObserver((entries) => {
  for (const en of entries) {
    if (!en.isIntersecting) continue;
    posterObserver.unobserve(en.target);
    const path = en.target.dataset.path;
    if (posters.has(path)) continue;
    fetchPoster(path).then((url) => {
      posters.set(path, url);
      document.querySelectorAll(`[data-path="${CSS.escape(path)}"] .poster img, .detail-poster[data-path="${CSS.escape(path)}"] img`)
        .forEach((img) => (img.src = url));
    }).catch(() => posters.set(path, ""));
  }
}, { rootMargin: "300px" });
function observePosters() {
  document.querySelectorAll("#grid .card").forEach((c) => {
    if (!posters.has(c.dataset.path)) posterObserver.observe(c);
  });
}

// ---------- movie details ----------
// The poster flies from the shelf into the details view (the app's one deliberate motion moment).
// With "reduce motion" on, or without View Transitions, the view simply appears.
let detailMovie = null;
let detailCard = null;
const infoCache = new Map();
const backdrops = new Map();

function withTransition(fn, from, to) {
  if (!document.startViewTransition || reduceMotion || !from) { fn(); return; }
  from.style.viewTransitionName = "poster";
  const t = document.startViewTransition(() => {
    from.style.viewTransitionName = "";
    fn();
    const target = to();
    if (target) target.style.viewTransitionName = "poster";
  });
  t.finished.finally(() => {
    $("detailPoster").style.viewTransitionName = "";
    $("showPoster").style.viewTransitionName = "";
    document.querySelectorAll("#grid .poster").forEach((p) => (p.style.viewTransitionName = ""));
  });
}

function cardFor(path) {
  return document.querySelector(`#grid .card[data-path="${CSS.escape(path)}"]`);
}

function openDetail(m, card) {
  detailCard = card;
  withTransition(() => {
    fillDetail(m);
    $("detail").showModal();
    $("detailBack").focus();
  }, card && card.querySelector(".poster"), () => $("detailPoster"));
}

function closeDetail() {
  if (!$("detail").open) return;
  const card = (detailCard && document.contains(detailCard) && detailCard) || (detailMovie && cardFor(detailMovie.path));
  withTransition(() => { $("detail").close(); detailMovie = null; }, $("detailPoster"), () => card && card.querySelector(".poster"));
  if (card) { card.tabIndex = 0; card.focus({ preventScroll: true }); }
}
$("detailBack").addEventListener("click", closeDetail);
$("detail").addEventListener("cancel", (e) => { e.preventDefault(); closeDetail(); });
$("dShow").addEventListener("click", () => detailMovie && api().ShowInFolder(detailMovie.path));
$("dPlay").addEventListener("click", () => detailMovie && playOnPC(detailMovie));
$("dKind").addEventListener("click", async () => {
  const m = detailMovie;
  if (!m) return;
  try {
    await api().SetKind([m.path], m.kind === "episode" ? "movie" : "show");
    closeDetail();
    await loadMovies();
    toast(m.kind === "episode" ? "Now shown as a movie." : "Now shown with your TV shows.");
  } catch (e) { toast(errText(e), true); }
});

// ---------- TV shows ----------
let showOpen = null;
let showSeason = 1;
const showMetaCache = new Map(); // show key -> TMDB info
const seasonCache = new Map();   // "key:season" -> TMDB episodes
const stills = new Map();        // TMDB picture path -> data: URL

function openShow(sh, card) {
  detailCard = card;
  showSeason = sh.seasons[0];
  withTransition(() => {
    fillShow(sh);
    $("showPage").showModal();
    $("showBack").focus();
  }, card && card.querySelector(".poster"), () => $("showPoster"));
}
function closeShow() {
  if (!$("showPage").open) return;
  const card = showOpen && cardFor(showOpen.id);
  withTransition(() => { $("showPage").close(); showOpen = null; }, $("showPoster"), () => card && card.querySelector(".poster"));
  if (card) { card.tabIndex = 0; card.focus({ preventScroll: true }); }
}
$("showBack").addEventListener("click", closeShow);
$("showPage").addEventListener("cancel", (e) => { e.preventDefault(); closeShow(); });
$("sShow").addEventListener("click", () => showOpen && api().ShowInFolder(showOpen.episodes[0].path));
$("sNotShow").addEventListener("click", async () => {
  const sh = showOpen;
  if (!sh) return;
  try {
    await api().SetKind(sh.episodes.map((e) => e.path), "movie");
    closeShow();
    await loadMovies();
    toast("Now shown as movies. To undo, open one and choose “This is a TV episode”.", false, 9000);
  } catch (e) { toast(errText(e), true); }
});

function loadSeason(sh, n) {
  const k = `${sh.key}:${n}`;
  if (seasonCache.has(k)) return Promise.resolve(seasonCache.get(k));
  if (!tmdbOn()) return Promise.resolve([]);
  return api().GetSeason(sh.title, n).then((eps) => { seasonCache.set(k, eps || []); return eps || []; })
    .catch(() => { seasonCache.set(k, []); return []; });
}
function tmdbEpisode(m) {
  return (seasonCache.get(`${m.showKey}:${m.season}`) || []).find((x) => x.episode === m.episode);
}
// TMDB's name for the episode, else the one in the file name, else "Episode 3".
function epName(m) {
  const t = tmdbEpisode(m);
  return (t && t.name) || m.epTitle || `Episode ${m.episode}`;
}
function loadStill(path, box) {
  const put = (url) => {
    if (!url) return;
    const img = el("img", { alt: "", decoding: "async" });
    img.addEventListener("load", () => img.classList.add("loaded"));
    img.src = url;
    box.replaceChildren(img);
  };
  if (stills.has(path)) { put(stills.get(path)); return; }
  api().GetEpisodeStill(path).then((u) => { stills.set(path, u); put(u); }).catch(() => stills.set(path, ""));
}

function fillShow(sh) {
  const same = $("showPage").open && showOpen && showOpen.key === sh.key;
  showOpen = sh;
  if (!sh.seasons.includes(showSeason)) showSeason = sh.seasons[0];
  const { name, year } = splitTitle(sh.title);
  $("showTitle").textContent = name;
  $("sYear").textContent = year;
  const nSeasons = sh.seasons.filter((n) => n > 0).length;
  $("sSeasonCount").textContent = nSeasons > 1 ? `${nSeasons} seasons` : "";
  const nEp = uniqueEpisodes(sh);
  $("sEpCount").textContent = `${nEp} episode${nEp === 1 ? "" : "s"}`;
  $("sSize").textContent = gb(sh.sizeMB);
  const dp = $("showPoster");
  dp.dataset.path = sh.id;
  dp.replaceChildren(...posterEl(sh).childNodes);
  if (!same) {
    for (const id of ["sRating", "sGenres"]) $(id).textContent = "";
    $("sOverview").classList.add("hidden");
    $("sSource").classList.add("hidden");
    $("showBg").classList.remove("loaded");
    $("showBg").style.backgroundImage = "";
  }
  fillShowMeta(sh);
  renderSeasonChips(sh);
  renderEpisodes(sh);
  renderShowSubs(sh);
  renderExtras($("sExtras"), $("sExtrasSection"), sh.extras);
  $("sTv").textContent = state.sharing
    ? `In VLC, go to Browse, then Local network, then “${state.settings.name}”.`
    : "Turn on Share with TV, then find it in VLC under Local network.";

  const showBg = (url) => {
    if (!url || !showOpen || showOpen.key !== sh.key) return;
    $("showBg").style.backgroundImage = `url("${url}")`;
    $("showBg").classList.add("loaded");
  };
  if (backdrops.has(sh.id)) showBg(backdrops.get(sh.id));
  else api().GetShowBackdrop(sh.title, sh.episodes[0].path).then((u) => { backdrops.set(sh.id, u); showBg(u); }).catch(() => backdrops.set(sh.id, ""));
}

function fillShowMeta(sh) {
  if (!tmdbOn()) return;
  const show = (meta) => {
    if (!showOpen || showOpen.key !== sh.key) return;
    const src = $("sSource");
    src.replaceChildren();
    src.classList.remove("hidden");
    const link = (text, fn) => el("button", { type: "button", class: "link-inline", text, onclick: fn });
    if (meta.id) {
      $("sRating").textContent = meta.votes >= 10 && meta.rating ? `Rated ${meta.rating.toFixed(1)}` : "";
      $("sGenres").textContent = (meta.genres || []).slice(0, 3).join(", ");
      $("sOverview").textContent = meta.overview || "";
      $("sOverview").classList.toggle("hidden", !meta.overview);
      src.append(document.createTextNode(meta.manual ? "Poster and info from TMDB (your pick). " : "Poster and info from TMDB. "),
        link("Not this show?", () => openMatch(sh)));
    } else if (meta.manual) {
      src.append(document.createTextNode("Using a frame from an episode. "),
        link("Look it up on TMDB again", async () => { await api().SetTMDBShowMatch(sh.title, 0); resetShow(sh); }));
    } else {
      src.append(document.createTextNode("Not found on TMDB. "), link("Find it", () => openMatch(sh)));
    }
  };
  if (showMetaCache.has(sh.key)) { show(showMetaCache.get(sh.key)); return; }
  api().GetShowMeta(sh.title).then((meta) => { showMetaCache.set(sh.key, meta); show(meta); }).catch(() => {});
}

// The show's TMDB match changed: start over for it.
function resetShow(sh) {
  posters.delete(sh.id); backdrops.delete(sh.id); showMetaCache.delete(sh.key);
  for (const k of [...seasonCache.keys()]) if (k.startsWith(sh.key + ":")) seasonCache.delete(k);
  for (const e of sh.episodes) { posters.delete(e.path); backdrops.delete(e.path); }
  fetchPoster(sh.id).then((url) => {
    posters.set(sh.id, url);
    document.querySelectorAll(`[data-path="${CSS.escape(sh.id)}"] .poster img, .detail-poster[data-path="${CSS.escape(sh.id)}"] img`)
      .forEach((img) => { img.classList.remove("loaded"); img.src = url; });
  }).catch(() => posters.set(sh.id, ""));
  if (showOpen && showOpen.key === sh.key) { $("showBg").classList.remove("loaded"); fillShow(sh); }
}

const seasonName = (n) => (n === 0 ? "Specials" : `Season ${n}`);

function renderSeasonChips(sh) {
  const box = $("sSeasons");
  box.replaceChildren();
  if (sh.seasons.length < 2) return;
  for (const n of sh.seasons) {
    box.append(el("button", {
      type: "button", class: "chip", "aria-pressed": n === showSeason ? "true" : "false", text: seasonName(n),
      onclick: () => { showSeason = n; renderSeasonChips(sh); renderEpisodes(sh); },
    }));
  }
}

function renderEpisodes(sh) {
  const list = $("sEpisodes");
  list.replaceChildren();
  const n = showSeason;
  const eps = sh.episodes.filter((e) => e.season === n);
  // Two copies of one episode (720p and 1080p, say) would look identical: label them.
  const seen = new Map();
  for (const e of eps) seen.set(e.episode, (seen.get(e.episode) || 0) + 1);
  for (const ep of eps) {
    const tm = tmdbEpisode(ep);
    const copy = seen.get(ep.episode) > 1 ? copyLabel(ep) : null;
    // A TMDB picture when there is one; no empty box otherwise.
    const still = tm && tm.stillPath ? el("div", { class: "ep-still" }) : null;
    if (still) loadStill(tm.stillPath, still);
    const status = ep.hasSubs ? el("span", { class: "ok" }, icon("i-cc"), document.createTextNode("Subtitles built in"))
      : canGetSubs(ep) ? el("span", { class: "srt", text: "Subtitles can be added" })
        : el("span", { text: "No subtitles" });
    const mins = tm && tm.minutes ? el("span", { text: fmtRuntime(tm.minutes) }) : null;
    const open = el("button", { type: "button", class: "ep-open", "aria-label": `Episode ${ep.episode}: ${epName(ep)}` },
      still,
      el("span", { class: "ep-text" },
        el("strong", {}, el("span", { class: "ep-num", text: String(ep.episode) }), el("span", { text: epName(ep), "data-no-tr": true })),
        el("small", { class: "ep-sub" }, copy, mins, status)));
    open.addEventListener("click", () => { cardClickAt = Date.now(); openEpisode(ep); });
    const row = el("li", { class: "ep", "data-path": ep.path }, open);
    if (preparing === ep.path) {
      row.append(el("div", { class: "ep-busy" }, el("div", { class: "ring" }, el("span", { text: "0%" })), el("span", { class: "stage", text: "Adding subtitles…" })));
    } else {
      row.append(el("button", { type: "button", class: "icon-btn ep-play", title: "Play on this PC", "aria-label": "Play on this PC", onclick: () => playOnPC(ep) }, icon("i-play")));
    }
    list.append(row);
  }
  if (tmdbOn() && !seasonCache.has(`${sh.key}:${n}`)) {
    loadSeason(sh, n).then(() => { if (showOpen && showOpen.key === sh.key && showSeason === n) renderEpisodes(showOpen); });
  }
}

// "1080p · 1.4 GB": the size right away, the picture quality once the file has been read.
function copyLabel(ep) {
  const tag = el("span", { class: "ep-copy", "data-no-tr": true });
  const fill = (i) => { tag.textContent = [i && i.resolution, gb(ep.sizeMB)].filter(Boolean).join(" · "); };
  const info = infoCache.get(ep.path);
  fill(info);
  if (!info) api().GetMovieInfo(ep.path).then((i) => { infoCache.set(ep.path, i); fill(i); }).catch(() => {});
  return tag;
}

// Episodes counted once each, however many copies of a file there are.
const uniqueEpisodes = (sh) => new Set(sh.episodes.map((e) => `${e.season}:${e.episode}`)).size;

function openEpisode(ep) {
  detailCard = null;
  fillDetail(ep);
  $("detail").showModal();
  $("detailBack").focus();
}

function fillEpisodeMeta(m) {
  const sh = shows.get(m.showKey);
  const show = () => {
    if (!detailMovie || detailMovie.path !== m.path) return;
    const t = tmdbEpisode(m);
    $("detailTitle").textContent = epName(m);
    if (t && t.minutes && !$("dRuntime").textContent) $("dRuntime").textContent = fmtRuntime(t.minutes);
    $("dOverview").textContent = (t && t.overview) || "";
    $("dOverview").classList.toggle("hidden", !(t && t.overview));
    const src = $("dSource");
    src.replaceChildren(document.createTextNode(t ? "Episode info from TMDB." : ""));
    src.classList.toggle("hidden", !t);
  };
  if (sh) loadSeason(sh, m.season).then(show); else show();
}

function renderExtras(list, section, extras) {
  list.replaceChildren();
  section.classList.toggle("hidden", !extras.length);
  for (const x of extras) {
    list.append(el("li", {}, el("button", { type: "button", class: "extra", title: "Play on this PC", onclick: () => playOnPC(x) },
      icon("i-play"), el("span", { text: x.title, "data-no-tr": true }), el("small", { class: "muted", text: gb(x.sizeMB) }))));
  }
}

// ---------- subtitles for every episode ----------
let batch = null; // { key, done, total, stop }
function renderShowSubs(sh) {
  const box = $("sSubs");
  box.replaceChildren();
  const todo = sh.episodes.filter(canGetSubs);
  const running = batch && batch.key === sh.key;
  $("sSubsSection").classList.toggle("hidden", !todo.length && !running);
  if (running) {
    const stop = el("button", { type: "button", class: "link-btn", text: batch.stop ? "Stopping after this episode…" : "Stop after this episode", disabled: batch.stop });
    stop.addEventListener("click", () => { batch.stop = true; stop.disabled = true; stop.textContent = "Stopping after this episode…"; });
    box.append(el("div", { class: "batch-line" }, el("span", { text: `Adding subtitles to episode ${Math.min(batch.done + 1, batch.total)} of ${batch.total}…` }), stop),
      el("p", { class: "muted small", text: "You can keep using the app. Each episode takes a minute or two." }));
    return;
  }
  if (!todo.length) return;
  const langs = new Map();
  for (const ep of todo) for (const s of ep.subs) { const k = s.lang || s.label; if (!langs.has(k)) langs.set(k, s.lang ? langName(s.lang, s.label) : s.label); }
  const sel = el("select", { "aria-label": "Subtitle language for every episode" });
  for (const [k, label] of langs) sel.append(el("option", { value: k, text: label, selected: k === state.settings.subLang }));
  const n = todo.length;
  box.append(
    el("p", { text: n === 1 ? "1 episode has subtitle files next to it. TVs ignore separate files, so add them into the episode." : `${n} episodes have subtitle files next to them. TVs ignore separate files, so add them into the episodes, one after another.` }),
    el("div", { class: "sub-row" }, sel, el("button", {
      class: "btn btn-primary", disabled: preparing !== null, onclick: () => addAllEpisodes(sh, sel.value),
    }, icon("i-plus"), document.createTextNode(n === 1 ? "Add to 1 episode" : `Add to all ${n} episodes`))));
}
function langName(code, fallback) {
  const l = (window.__langs || []).find(([c]) => c === code);
  return l ? l[1] : fallback.replace(/\s*\(.*\)$/, "");
}

async function addAllEpisodes(sh, lang) {
  if (preparing || batch) return;
  batch = { key: sh.key, done: 0, total: sh.episodes.filter(canGetSubs).length, stop: false };
  const tried = new Set();
  let added = 0;
  try {
    while (!batch.stop) {
      const cur = shows.get(sh.key);
      const ep = cur && cur.episodes.find((e) => canGetSubs(e) && !tried.has(e.path));
      if (!ep) break;
      tried.add(ep.path);
      const sub = ep.subs.find((s) => (s.lang || s.label) === lang) || ep.subs[ep.defaultSub];
      if (showOpen && showOpen.key === sh.key) renderShowSubs(cur);
      const r = await addSubtitles(ep, sub.path, true);
      if (r === "cancelled") break;
      if (r === "ok") added++;
      batch.done++;
    }
  } finally {
    batch = null;
    await loadMovies();
    if (added) toast(added === 1 ? "Subtitles added to 1 episode." : `Subtitles added to ${added} episodes.`, false, 7000);
  }
}

// ---------- play on this PC ----------
// Opens the movie in the PC's own video player. TV Share can't see inside that player, so unless it
// already is VLC, the message offers VLC: it plays every format here and shows built-in subtitles.
function vlcAction(r, m) {
  return r.hasVLC
    ? el("button", { type: "button", class: "toast-action", text: "Play in VLC", onclick: async (e) => {
        e.currentTarget.closest(".toast")?.remove();
        try { await api().PlayInVLC(m.path); } catch (err) { toast(errText(err), true); }
      } })
    : el("button", { type: "button", class: "toast-action", text: "Get VLC (free)", onclick: () => api().GetVLCPage() });
}
let lastPlay = 0, cardClickAt = 0;
// Double-clicking a poster: the first click opens the full-window page (with a transition, during
// which clicks go to the document itself), so catch the second click wherever it lands.
const secondClick = (e) => {
  if (e.detail >= 2 && Date.now() - cardClickAt < 800 && detailMovie) playOnPC(detailMovie);
};
document.addEventListener("click", secondClick, true);
document.addEventListener("dblclick", secondClick, true);
async function playOnPC(m) {
  if (Date.now() - lastPlay < 1500) return; // a double-click on the button shouldn't open it twice
  lastPlay = Date.now();
  let r = { player: "", isVLC: false, hasVLC: false };
  try {
    r = await api().PlayOnPC(m.path);
  } catch (e) {
    const msg = errText(e);
    toast(/[.!?]$/.test(msg) ? msg : msg + ".", true, 12000, vlcAction(r, m));
    return;
  }
  const opening = r.player ? `Opening in ${r.player}.` : "Opening in your video player.";
  if (r.isVLC) { toast(opening); return; }
  toast(opening, false, 12000, el("span", { text: "Won't play, or no subtitles?" }), vlcAction(r, m));
}

function fillMeta(m) {
  if (!tmdbOn()) return;
  if (m.kind === "episode") { fillEpisodeMeta(m); return; }
  const show = (meta) => {
    if (!detailMovie || detailMovie.path !== m.path) return;
    const src = $("dSource");
    src.replaceChildren();
    src.classList.remove("hidden");
    const link = (text, fn) => el("button", { type: "button", class: "link-inline", text, onclick: fn });
    if (meta.id) {
      $("dRating").textContent = meta.votes >= 10 && meta.rating ? `Rated ${meta.rating.toFixed(1)}` : "";
      $("dRating").title = meta.votes ? `TMDB rating from ${meta.votes} votes` : "";
      $("dGenres").textContent = (meta.genres || []).slice(0, 3).join(", ");
      $("dOverview").textContent = meta.overview || "";
      $("dOverview").classList.toggle("hidden", !meta.overview);
      src.append(document.createTextNode(meta.manual ? "Poster and info from TMDB (your pick). " : "Poster and info from TMDB. "),
        link("Not this movie?", () => openMatch(m)));
    } else if (meta.manual) {
      src.append(document.createTextNode("Using a frame from the movie. "),
        link("Look it up on TMDB again", async () => { await api().SetTMDBMatch(m.path, 0); resetOne(m.path); }));
    } else {
      src.append(document.createTextNode("Not found on TMDB. "), link("Find it", () => openMatch(m)));
    }
  };
  if (metaCache.has(m.path)) { show(metaCache.get(m.path)); return; }
  api().GetMovieMeta(m.path).then((meta) => { metaCache.set(m.path, meta); show(meta); })
    .catch(() => { /* offline: the frame poster and file details still show */ });
}

const fmtRuntime = (min) => (min >= 60 ? `${Math.floor(min / 60)} h ${String(min % 60).padStart(2, "0")} min` : `${min} min`);

function fillDetail(m, force = false) {
  const same = !force && detailMovie && detailMovie.path === m.path;
  detailMovie = m;
  const { name, year } = splitTitle(m.title);
  const isEp = m.kind === "episode";
  $("detailTitle").textContent = isEp ? epName(m) : name;
  $("dEyebrow").classList.toggle("hidden", !isEp);
  if (isEp) {
    $("dEyeShow").textContent = splitTitle(m.show).name;
    $("dEyeEp").textContent = m.season === 0 ? `Specials, Episode ${m.episode}` : `Season ${m.season}, Episode ${m.episode}`;
  }
  $("detailBack").setAttribute("aria-label", isEp ? "Back to the show" : "Back to your movies");
  $("dYear").textContent = year;
  $("dSize").textContent = gb(m.sizeMB);
  renderExtras($("dExtras"), $("dExtrasSection"), isEp ? [] : (extrasFor.get(m.path) || []));
  $("dKind").textContent = isEp ? "Not part of a show? Show it as a movie" : "This is a TV episode";

  const dp = $("detailPoster");
  dp.dataset.path = m.path;
  dp.replaceChildren(...posterEl(m).childNodes);

  if (!same) {
    $("dRuntime").textContent = "";
    $("dVideo").textContent = "";
    $("dAudio").textContent = "Reading the file…";
    $("detailBg").classList.remove("loaded");
    $("detailBg").style.backgroundImage = "";
    $("dRating").textContent = "";
    $("dGenres").textContent = "";
    $("dOverview").classList.add("hidden");
    $("dSource").classList.add("hidden");
  }
  fillMeta(m);
  renderDetailSubs(m);
  $("dTv").textContent = m.outside
    ? "This file is outside your shared folder, so the TV can't see it. Move it into the shared folder to watch it."
    : state.sharing
      ? `In VLC, go to Browse, then Local network, then “${state.settings.name}”.`
      : "Turn on Share with TV, then find it in VLC under Local network.";

  const showInfo = (i) => {
    if (!detailMovie || detailMovie.path !== m.path) return;
    $("dRuntime").textContent = i.minutes ? fmtRuntime(i.minutes) : "";
    $("dVideo").textContent = [i.resolution, i.video].filter(Boolean).join(" ");
    $("dAudio").textContent = i.audio.length ? i.audio.join(", ") : "No sound track found";
    const built = $("dSubsBuilt");
    if (built && i.subtitles.length) built.textContent = "Languages: " + [...new Set(i.subtitles)].join(", ");
  };
  const info = infoCache.get(m.path);
  if (info) showInfo(info);
  else api().GetMovieInfo(m.path).then((i) => { infoCache.set(m.path, i); showInfo(i); })
    .catch(() => { if (detailMovie && detailMovie.path === m.path) $("dAudio").textContent = "Couldn't read this file's details."; });

  const showBg = (url) => {
    if (!url || !detailMovie || detailMovie.path !== m.path) return;
    $("detailBg").style.backgroundImage = `url("${url}")`;
    $("detailBg").classList.add("loaded");
  };
  if (backdrops.has(m.path)) showBg(backdrops.get(m.path));
  else api().GetBackdrop(m.path).then((u) => { backdrops.set(m.path, u); showBg(u); }).catch(() => backdrops.set(m.path, ""));
}

function renderDetailSubs(m) {
  const box = $("dSubs");
  box.replaceChildren();
  if (preparing === m.path) {
    box.append(el("div", { class: "inline-progress", "data-path": m.path },
      el("div", { class: "ring" }, el("span", { text: "0%" })),
      el("div", {}, el("p", { class: "stage", text: "Starting…" }),
        el("p", { class: "muted small", text: "You can keep using the app. Big movies take a minute or two." }),
        el("button", { class: "link-btn", style: "margin-left:0", text: "Cancel", onclick: () => api().CancelSubtitles() }))));
    return;
  }
  if (m.hasSubs) {
    box.append(el("div", { class: "sub-state" }, icon("i-check"), document.createTextNode("Built into the movie. The TV will show them.")),
      el("p", { id: "dSubsBuilt" }));
    const i = infoCache.get(m.path);
    if (i && i.subtitles.length) $("dSubsBuilt").textContent = "Languages: " + [...new Set(i.subtitles)].join(", ");
    const amount = el("select", { "aria-label": "How much to move the subtitles" });
    for (const s of [0.5, 1, 2, 5]) amount.append(el("option", { value: s * 1000, text: `${s} s`, selected: s === 1 }));
    const busyNow = preparing !== null;
    box.append(el("div", { class: "sync-box" },
      el("p", { class: "muted small", text: "Subtitles out of sync with the voices? TV Share can match them to the sound." }),
      el("div", { class: "sub-row" },
        el("button", { class: "btn btn-ghost", disabled: busyNow, onclick: () => fixSync(m, 0) }, icon("i-refresh"), document.createTextNode("Fix sync"))),
      el("div", { class: "sub-row nudge" },
        el("span", { class: "muted small", text: "Or move them yourself:" }), amount,
        el("button", { class: "btn btn-ghost", disabled: busyNow, onclick: () => fixSync(m, -Number(amount.value)) }, document.createTextNode("Earlier")),
        el("button", { class: "btn btn-ghost", disabled: busyNow, onclick: () => fixSync(m, Number(amount.value)) }, document.createTextNode("Later")))));
    return;
  }
  if (!m.subs.length) {
    box.append(el("p", { text: "None found on your PC yet." }));
    box.append(onlineBox(m));
    return;
  }
  box.append(el("p", { text: "Subtitle files were found next to the movie. TVs ignore separate files, so add them into the movie. It takes about a minute, with no quality loss." }));
  const sel = el("select", { "aria-label": `Subtitle language for ${m.title}` });
  m.subs.forEach((s, i) => sel.append(el("option", { value: s.path, text: s.label, selected: i === m.defaultSub })));
  box.append(el("div", { class: "sub-row" }, sel, el("button", {
    class: "btn btn-primary btn-lg", disabled: preparing !== null,
    onclick: () => addSubtitles(m, sel.value),
  }, icon("i-plus"), document.createTextNode("Add subtitles"))));
  box.append(onlineBox(m, true));
}

// ---------- subtitles from OpenSubtitles ----------
let osStatus = null;
async function refreshOSStatus() {
  try { osStatus = await api().GetOpenSubtitlesStatus(); } catch { osStatus = null; }
  renderOSSettings();
  return osStatus;
}

// "Find subtitles online": a language, a search, then a short list. Exact matches first.
function onlineBox(m, secondary = false) {
  const wrap = el("div", { class: "online-box" + (secondary ? " secondary" : "") });
  const lang = el("select", { "aria-label": "Subtitle language" });
  const langs = (window.__langs || [["eng", "English"]]);
  for (const [code, name] of langs) lang.append(el("option", { value: code, text: name, selected: code === state.settings.subLang }));
  const list = el("ul", { class: "os-list", "aria-live": "polite" });
  const go = el("button", { class: "btn btn-ghost", disabled: preparing !== null }, icon("i-search"), document.createTextNode("Find subtitles online"));
  go.addEventListener("click", () => searchOnline(m, lang.value, list, go));
  wrap.append(
    el("p", { class: "muted small", text: secondary ? "Or find other subtitles online:" : "Find subtitles on OpenSubtitles.com. Ones made for this exact file are in sync from the start." }),
    el("div", { class: "sub-row" }, lang, go),
    list,
    el("p", { class: "muted small os-privacy", text: "Searching sends this file's fingerprint and the movie's title and year to OpenSubtitles. Nothing else." }));
  return wrap;
}

async function searchOnline(m, lang, list, btn) {
  btn.disabled = true;
  list.replaceChildren(el("li", { class: "muted small", text: "Searching OpenSubtitles…" }));
  try {
    const found = await api().FindSubtitlesOnline(m.path, lang);
    list.replaceChildren();
    if (!found.length) {
      list.append(el("li", { class: "muted small", text: "Nothing found in this language. Try another one." }));
      return;
    }
    for (const r of found.slice(0, 8)) {
      const tags = [];
      if (r.exact) tags.push(el("span", { class: "os-exact" }, icon("i-check"), document.createTextNode("Made for this file")));
      if (r.hi) tags.push(el("span", { class: "os-tag", text: "For the hard of hearing" }));
      list.append(el("li", {}, el("button", {
        type: "button", class: "os-item", disabled: preparing !== null,
        onclick: () => getOnline(m, r),
      },
        el("span", { class: "os-main" },
          el("strong", { text: r.langLabel }),
          el("small", { class: "os-release", text: r.release || "Unknown release", "data-no-tr": true })),
        el("span", { class: "os-side" }, ...tags, el("small", { class: "muted", text: `${r.downloads.toLocaleString(window.i18n && window.i18n.lang === "fr" ? "fr-FR" : "en-US")} downloads` })))));
    }
  } catch (e) {
    list.replaceChildren(el("li", { class: "error small", text: errText(e) }));
    if (/sign in|downloads/i.test(errText(e))) list.append(el("li", {}, el("button", { type: "button", class: "link-btn", style: "margin-left:0", text: "Sign in or create a free account", onclick: openOSSettings })));
  } finally {
    btn.disabled = preparing !== null;
  }
}

async function getOnline(m, r) {
  try {
    toast("Downloading the subtitles…");
    const path = await api().DownloadSubtitleOnline(m.path, r.fileId, r.language);
    refreshOSStatus();
    await addSubtitles(m, path);
  } catch (e) {
    const msg = errText(e);
    toast(msg, true, 9000);
    if (/sign in/i.test(msg)) openOSSettings();
  }
}

function openOSSettings() {
  if ($("detail").open) closeDetail();
  $("settingsBtn").click();
  setTimeout(() => { const s = $("osSection"); if (s) s.scrollIntoView({ block: "start" }); }, 150);
}

// Settings: sign in, sign up, own key. Built so a non-technical person can do it in a minute.
function renderOSSettings() {
  const st = osStatus;
  const box = $("osAccount");
  if (!box) return;
  box.replaceChildren();
  if (!st || !st.available) {
    box.append(el("p", { class: "muted small", text: "To search for subtitles online, add a free OpenSubtitles API key below." }));
  } else if (st.user) {
    const left = st.remaining >= 0 ? `${st.remaining} downloads left today` : "Signed in";
    box.append(el("div", { class: "os-signed" },
      el("span", {}, el("strong", { text: st.user, "data-no-tr": true }), el("small", { class: "muted", text: left })),
      el("button", { type: "button", class: "btn btn-ghost", text: "Sign out", onclick: async () => { await api().SignOutOpenSubtitles(); refreshOSStatus(); } })));
  } else {
    const user = el("input", { class: "input", placeholder: "Username", autocomplete: "username", spellcheck: "false", "aria-label": "OpenSubtitles username" });
    const pass = el("input", { class: "input", type: "password", placeholder: "Password", autocomplete: "current-password", "aria-label": "OpenSubtitles password" });
    const err = el("p", { class: "error small hidden" });
    const signIn = el("button", { type: "button", class: "btn btn-primary", text: "Sign in" });
    signIn.addEventListener("click", async () => {
      signIn.disabled = true; err.classList.add("hidden");
      try {
        await api().SignInOpenSubtitles(user.value, pass.value);
        pass.value = "";
        toast("Signed in to OpenSubtitles.");
        refreshOSStatus();
      } catch (e) { err.textContent = errText(e); err.classList.remove("hidden"); }
      finally { signIn.disabled = false; }
    });
    pass.addEventListener("keydown", (e) => { if (e.key === "Enter") { e.preventDefault(); signIn.click(); } });
    box.append(
      el("p", { class: "muted small", text: "Without an account: 5 subtitle downloads a day. With a free account: more, on your own allowance." }),
      el("ol", { class: "os-steps" },
        el("li", {}, document.createTextNode("No account yet? "),
          el("button", { type: "button", class: "link-inline", text: "Create one for free", onclick: () => api().OpenSubtitlesSignupPage() }),
          document.createTextNode(" (about a minute, just an email and a password).")),
        el("li", { text: "Then sign in here. Your password is only sent to OpenSubtitles, never saved." })),
      el("div", { class: "os-login" }, user, pass, signIn), err);
  }
  $("osKeyNote").textContent = st && st.ownKey
    ? "Using your own key. Empty the box and press Save to go back to the app's key."
    : st && st.builtIn ? "Optional. This app already has one; use yours if you prefer."
      : "Needed to search for subtitles. Getting one is free.";
  $("osKeyInput").placeholder = st && st.ownKey ? "Your key is saved" : "Paste a key";
}

$("osKeySave").addEventListener("click", async () => {
  const btn = $("osKeySave");
  btn.disabled = true;
  try {
    const key = $("osKeyInput").value.trim();
    await api().SetOpenSubtitlesKey(key);
    $("osKeyInput").value = "";
    toast(key ? "Key saved. OpenSubtitles accepted it." : "Back to the app's own key.");
    refreshOSStatus();
  } catch (e) { toast(errText(e), true); }
  finally { btn.disabled = false; }
});
$("osKeyHelp").addEventListener("click", () => api().OpenSubtitlesKeyPage());

// ---------- adding subtitles ----------
function showBusy(poster) {
  if (poster.querySelector(".busy")) return;
  poster.append(el("div", { class: "busy" },
    el("div", { class: "ring" }, el("span", { text: "0%" })), el("small", { class: "stage", text: "Adding subtitles…" }),
    el("button", { class: "link-btn", text: "Cancel", onclick: (e) => { e.stopPropagation(); api().CancelSubtitles(); } })));
}

// quiet: part of "Add to all episodes", which says once at the end what it did.
async function addSubtitles(m, subPath, quiet = false) {
  preparing = m.path;
  document.querySelectorAll(".card-meta .add, #dSubs .btn-primary, #sSubs .btn-primary").forEach((b) => (b.disabled = true));
  const card = cardFor(m.path);
  if (card) showBusy(card.querySelector(".poster"));
  if (detailMovie && detailMovie.path === m.path) renderDetailSubs(m);
  if (showOpen && m.kind === "episode" && showOpen.key === m.showKey) renderEpisodes(showOpen);

  let result = "ok";
  try {
    const res = await api().AddSubtitles(m.path, subPath);
    if (!quiet) {
      toast(res.recycled
        ? `Subtitles added to “${itemName(m)}”. The original is in the Recycle Bin.`
        : `Subtitles added to “${itemName(m)}”.`);
      if (res.syncNote) toast(res.syncNote, false, 7000);
    }
    if (detailMovie && detailMovie.path === m.path && res.output) detailMovie = { ...detailMovie, path: res.output };
  } catch (e) {
    const msg = errText(e);
    result = msg === "cancelled" ? "cancelled" : "error";
    if (msg !== "cancelled") toast(quiet ? `${itemName(m)}: ${msg}` : msg, true, 7000);
  } finally {
    preparing = null;
    await loadMovies();
  }
  return result;
}

// Re-time the subtitles inside a movie: shiftMs 0 = match to the sound, otherwise move by that much.
async function fixSync(m, shiftMs) {
  preparing = m.path;
  document.querySelectorAll(".card-meta .add, #dSubs .btn").forEach((b) => (b.disabled = true));
  const card = cardFor(m.path);
  if (card) showBusy(card.querySelector(".poster"));
  if (detailMovie && detailMovie.path === m.path) renderDetailSubs(m);
  try {
    const res = await api().FixSubtitleSync(m.path, shiftMs);
    toast(res.syncNote || "Subtitles updated.", false, 7000);
    if (detailMovie && detailMovie.path === m.path && res.output) detailMovie = { ...detailMovie, path: res.output };
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
  rt().EventsOn("setup", (line) => {
    $("setupLog").textContent = line;
    if (/ffmpeg/i.test(line) && $("depRclone").classList.contains("spinning")) $("depRclone").className = "check done";
  });
  rt().EventsOn("network", (n) => { if (state) { state.network = n; render(); } });
  rt().EventsOn("share:started", async () => { state = await api().GetState(); render(); });
  rt().EventsOn("share:stopped", async (msg) => {
    state = await api().GetState();
    render();
    if (msg) showShareError("Sharing stopped: " + msg);
  });
  rt().EventsOn("prepare:progress", (d) => {
    document.querySelectorAll(`[data-path="${CSS.escape(d.path)}"] .ring`).forEach((ring) => {
      ring.style.setProperty("--p", d.percent.toFixed(1));
      ring.querySelector("span").textContent = Math.round(d.percent) + "%";
    });
    if (d.stage) document.querySelectorAll(`[data-path="${CSS.escape(d.path)}"] .stage`).forEach((s) => (s.textContent = d.stage));
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
