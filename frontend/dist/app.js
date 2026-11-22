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
  if ($("detail").open) return;
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
  posters.clear(); backdrops.clear(); metaCache.clear();
  renderMovies();
}
// One movie's match changed: refresh just that one.
function resetOne(path) {
  posters.delete(path); backdrops.delete(path); metaCache.delete(path);
  api().GetPoster(path).then((url) => {
    posters.set(path, url);
    document.querySelectorAll(`[data-path="${CSS.escape(path)}"] .poster img, #detailPoster[data-path="${CSS.escape(path)}"] img`)
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
  $("matchDlg").showModal();
  await runMatchSearch();
}
async function runMatchSearch() {
  const list = $("matchList");
  list.replaceChildren(el("li", { class: "muted small", text: "Searching TMDB…" }));
  try {
    const found = await api().FindTMDBMatches(matchFor.path, $("matchQuery").value);
    const cur = (metaCache.get(matchFor.path) || {}).id;
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
    await api().SetTMDBMatch(m.path, id);
    resetOne(m.path);
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
    const hit = (!q || c.dataset.title.includes(q)) && (filter === "all" || c.dataset.subs === "1");
    c.classList.toggle("hidden", !hit);
    if (hit) shown++;
  });
  $("noMatch").textContent = q ? "No movie matches your search." : "Every movie already has its subtitles.";
  $("noMatch").classList.toggle("hidden", !(movies.length && shown === 0));
  fixRovingFocus();
}

function sortedMovies() {
  const list = movies.slice();
  if (sortBy === "added") list.sort((a, b) => b.added - a.added);
  else if (sortBy === "size") list.sort((a, b) => b.sizeMB - a.sizeMB);
  return list; // "title": the app already sends them A to Z
}

function renderMovies() {
  const grid = $("grid");
  grid.replaceChildren();
  $("emptyState").classList.toggle("hidden", movies.length > 0);
  $("shelfTools").classList.toggle("hidden", movies.length === 0);
  const canAdd = movies.filter(canGetSubs).length;
  $("countAll").textContent = movies.length;
  $("countSubs").textContent = canAdd;
  $("moviesSub").textContent = movies.length === 0
    ? "Nothing found yet"
    : `${movies.length} movie${movies.length > 1 ? "s" : ""}` + (canAdd ? `, ${canAdd} can get subtitles` : "");
  if (!canAdd && filter === "subs") document.querySelector('.chip[data-filter="all"]').click();

  for (const m of sortedMovies()) grid.append(movieCard(m));
  applyView();
  observePosters();
  if (detailMovie) {
    const fresh = movies.find((m) => m.path === detailMovie.path) || movies.find((m) => m.title === detailMovie.title);
    if (fresh) fillDetail(fresh); else closeDetail();
  }
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
  const cached = posters.get(m.path);
  if (cached) img.src = cached;
  if (m.hasSubs) poster.append(el("span", { class: "corner corner-ok", title: "Subtitles built in" }, icon("i-cc"), document.createTextNode("CC")));
  else if (m.subs.length) poster.append(el("span", { class: "corner", title: "Subtitles can be added" }, document.createTextNode("SRT")));
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
    class: "card", tabindex: "-1", "data-path": m.path, "data-title": m.title.toLowerCase(), "data-subs": canGetSubs(m) ? "1" : "0",
    "aria-label": `${m.title}, ${gb(m.sizeMB)}, ${statusText}. Press Enter for details.`,
  }, poster, el("div", { class: "card-title", text: name, title: m.path, "data-no-tr": true }), meta);
  card.addEventListener("click", () => openDetail(m, card));
  card.addEventListener("keydown", (e) => {
    if (e.target === card && (e.key === "Enter" || e.key === " ")) { e.preventDefault(); openDetail(m, card); }
  });
  if (preparing === m.path) showBusy(poster);
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
    api().GetPoster(path).then((url) => {
      posters.set(path, url);
      document.querySelectorAll(`[data-path="${CSS.escape(path)}"] .poster img, #detailPoster[data-path="${CSS.escape(path)}"] img`)
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

function fillMeta(m) {
  if (!tmdbOn()) return;
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
  $("detailTitle").textContent = name;
  $("dYear").textContent = year;
  $("dSize").textContent = gb(m.sizeMB);

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

async function addSubtitles(m, subPath) {
  preparing = m.path;
  document.querySelectorAll(".card-meta .add, #dSubs .btn-primary").forEach((b) => (b.disabled = true));
  const card = cardFor(m.path);
  if (card) showBusy(card.querySelector(".poster"));
  if (detailMovie && detailMovie.path === m.path) renderDetailSubs(m);

  try {
    const res = await api().AddSubtitles(m.path, subPath);
    toast(res.recycled
      ? `Subtitles added to “${m.title}”. The original is in the Recycle Bin.`
      : `Subtitles added to “${m.title}”.`);
    if (res.syncNote) toast(res.syncNote, false, 7000);
    if (detailMovie && detailMovie.path === m.path && res.output) detailMovie = { ...detailMovie, path: res.output };
  } catch (e) {
    const msg = errText(e);
    if (msg !== "cancelled") toast(msg, true, 7000);
  } finally {
    preparing = null;
    loadMovies();
  }
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
