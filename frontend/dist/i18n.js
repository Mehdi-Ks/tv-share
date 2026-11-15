"use strict";
// Translations. English is written in the page and the code; this file swaps in another
// language as text reaches the screen (including messages from the Go side), so nothing
// else in the app has to change. To add a language: copy the FR block, translate the
// right-hand side, and add it to LANGS.
//
// Rules for translators: keep {placeholders} and HTML tags as they are; a key that is
// missing simply stays in English.

(() => {
  const FR = {
    // ---- Top bar and banners
    "Search your movies": "Rechercher dans vos films",
    "Share with TV": "Partager avec la TV",
    "Off": "Désactivé",
    "Starting…": "Démarrage…",
    "Stop sharing": "Arrêter le partage",
    "Let the TV see your movies": "Laisser la TV voir vos films",
    "How to watch on your TV": "Comment regarder sur votre TV",
    "Settings": "Paramètres",
    "Open network settings": "Ouvrir les paramètres réseau",
    "Official posters": "Affiches officielles",
    "Use official posters": "Utiliser les affiches officielles",
    "Keep movie frames": "Garder les images des films",

    // ---- First run: helpers
    "Install helpers": "Installer les outils",
    "Installing…": "Installation…",
    "Shows your movie folder to the TV": "Montre votre dossier de films à la TV",
    "Posters, and subtitles the TV can show": "Affiches, et sous-titres lisibles par la TV",
    "Checking the latest rclone…": "Recherche de la dernière version de rclone…",
    "Checking the latest FFmpeg…": "Recherche de la dernière version de FFmpeg…",

    // ---- First run: folder and name
    "Choose another folder": "Choisir un autre dossier",
    "Name on the TV": "Nom sur la TV",
    "You'll pick this name on the TV, so make it easy to spot.": "C'est ce nom que vous choisirez sur la TV : rendez-le facile à repérer.",
    "Posters": "Affiches",
    "You can change this any time in Settings.": "Vous pourrez changer ce choix à tout moment dans les Paramètres.",
    "Plus plot and rating. Looks each title up on TMDB (only the title and year are sent).": "Avec résumé et note. Chaque titre est cherché sur TMDB (seuls le titre et l'année sont envoyés).",
    "Movie frames": "Images des films",
    "A frame from each movie. Nothing is looked up online.": "Une image tirée de chaque film. Rien n'est cherché en ligne.",
    "Continue": "Continuer",
    "Looking for movies…": "Recherche des films…",
    "This folder doesn't exist. Choose another one.": "Ce dossier n'existe pas. Choisissez-en un autre.",
    "No movies here yet. That's fine, you can add them later.": "Pas encore de films ici. Ce n'est pas grave, vous pourrez en ajouter plus tard.",
    "Choose how your posters should look.": "Choisissez l'apparence de vos affiches.",
    "All set. Turn on Share with TV to show your movies on the TV.": "Tout est prêt. Activez « Partager avec la TV » pour voir vos films sur la TV.",

    // ---- Shelf
    "Your movies": "Vos films",
    "Refresh": "Actualiser",
    "Look for new movies": "Chercher de nouveaux films",
    "Open folder": "Ouvrir le dossier",
    "Show": "Afficher",
    "All": "Tous",
    "Can get subtitles": "Sous-titres à ajouter",
    "Sort": "Trier",
    "Recently added": "Ajoutés récemment",
    "A to Z": "De A à Z",
    "Biggest first": "Les plus gros d'abord",
    "No movies here yet": "Pas encore de films ici",
    "Open the folder": "Ouvrir le dossier",
    "No movie matches your search.": "Aucun film ne correspond à votre recherche.",
    "Every movie already has its subtitles.": "Tous les films ont déjà leurs sous-titres.",
    "Nothing found yet": "Rien trouvé pour l'instant",
    "Subtitles built in": "Sous-titres intégrés",
    "Subtitles can be added": "Sous-titres à ajouter",
    "Subtitles": "Sous-titres",
    "Not shared": "Non partagé",
    "Outside your shared folder: the TV won't see it": "En dehors du dossier partagé : la TV ne le verra pas",
    "Adding subtitles…": "Ajout des sous-titres…",
    "Cancel": "Annuler",
    "Drop to add this movie": "Déposez pour ajouter ce film",
    "Drop a movie, or a folder with a movie inside.": "Déposez un film, ou un dossier qui contient un film.",

    // ---- Help panel
    "Watch on your TV": "Regarder sur la TV",
    "Close": "Fermer",
    "your PC": "votre PC",
    "More fixes for common problems": "Autres solutions aux problèmes courants",

    // ---- Details
    "Back to your movies": "Retour à vos films",
    "Back (Esc)": "Retour (Échap)",
    "Sound": "Son",
    "On the TV": "Sur la TV",
    "Show in folder": "Afficher dans le dossier",
    "Reading the file…": "Lecture du fichier…",
    "No sound track found": "Aucune piste audio trouvée",
    "Couldn't read this file's details.": "Impossible de lire les détails de ce fichier.",
    "Built into the movie. The TV will show them.": "Intégrés au film. La TV les affichera.",
    "None found. Put a .srt file next to the movie (same name, or in a Subs folder), then press Refresh.": "Aucun trouvé. Placez un fichier .srt à côté du film (même nom, ou dans un dossier Subs), puis cliquez sur Actualiser.",
    "Subtitle files were found next to the movie. TVs ignore separate files, so add them into the movie. It takes about a minute, with no quality loss.": "Des fichiers de sous-titres ont été trouvés à côté du film. Les TV ignorent les fichiers séparés : ajoutez-les dans le film. Cela prend environ une minute, sans perte de qualité.",
    "Add subtitles": "Ajouter les sous-titres",
    "Building the subtitles in…": "Intégration des sous-titres…",
    "Finishing the file…": "Finalisation du fichier…",
    "Checking the new file…": "Vérification du nouveau fichier…",
    "Moving the original to the Recycle Bin…": "Envoi de l'original à la Corbeille…",
    "Done": "Terminé",
    "You can keep using the app. Big movies take a minute or two.": "Vous pouvez continuer à utiliser l'app. Les gros films prennent une à deux minutes.",
    "This file is outside your shared folder, so the TV can't see it. Move it into the shared folder to watch it.": "Ce fichier est en dehors du dossier partagé : la TV ne peut pas le voir. Déplacez-le dans le dossier partagé pour le regarder.",
    "Turn on Share with TV, then find it in VLC under Local network.": "Activez « Partager avec la TV », puis trouvez-le dans VLC sous Réseau local.",
    "Not this movie?": "Ce n'est pas le bon film ?",
    "Find it": "Le trouver",
    "Look it up on TMDB again": "Le chercher à nouveau sur TMDB",
    "Poster and info from TMDB.": "Affiche et infos de TMDB.",
    "Poster and info from TMDB (your pick).": "Affiche et infos de TMDB (votre choix).",
    "Using a frame from the movie.": "Image tirée du film.",
    "Not found on TMDB.": "Introuvable sur TMDB.",
    "Subtitles (same name as the movie)": "Sous-titres (même nom que le film)",

    // ---- Firewall heads-up
    "One question from Windows": "Une question de Windows",
    "Not now": "Pas maintenant",
    "Got it, start sharing": "Compris, lancer le partage",

    // ---- "Not this movie?"
    "Which movie is it?": "De quel film s'agit-il ?",
    "Search TMDB by title": "Chercher un titre sur TMDB",
    "Search": "Rechercher",
    "Searching TMDB…": "Recherche sur TMDB…",
    "Nothing found. Try another spelling, or leave out the year.": "Rien trouvé. Essayez une autre orthographe, ou sans l'année.",
    "Use a frame from the movie instead": "Utiliser plutôt une image du film",
    "Results from TMDB.": "Résultats de TMDB.",
    "Year unknown": "Année inconnue",

    // ---- Settings
    "Close settings": "Fermer les paramètres",
    "Close (Esc)": "Fermer (Échap)",
    "Your share": "Votre partage",
    "Rename": "Renommer",
    "Movie folder": "Dossier des films",
    "Change": "Changer",
    "Network": "Réseau",
    "Checking…": "Vérification…",
    "Posters and movie info": "Affiches et infos des films",
    "Official posters from TMDB": "Affiches officielles de TMDB",
    "Posters, plot and rating from TMDB. Only movie titles and years are sent. Off: a frame from each movie, nothing online.": "Affiches, résumé et note depuis TMDB. Seuls les titres et années des films sont envoyés. Désactivé : une image de chaque film, rien en ligne.",
    "Your own TMDB key": "Votre propre clé TMDB",
    "Optional. This app already has one; use yours if you prefer.": "Facultatif. L'app en a déjà une ; utilisez la vôtre si vous préférez.",
    "Using your own key. Empty the box and press Save to go back to the app's key.": "Votre clé est utilisée. Videz le champ et cliquez sur Enregistrer pour revenir à celle de l'app.",
    "Needed for official posters. Get a free key at themoviedb.org (Settings, API).": "Nécessaire pour les affiches officielles. Obtenez une clé gratuite sur themoviedb.org (Paramètres, API).",
    "Paste a key": "Collez une clé",
    "Your key is saved": "Votre clé est enregistrée",
    "Save": "Enregistrer",
    "Key saved. TMDB accepted it.": "Clé enregistrée. TMDB l'a acceptée.",
    "Back to the app's own key.": "Retour à la clé de l'app.",
    "Options": "Options",
    "Start sharing when the app opens": "Partager dès l'ouverture de l'app",
    "Saves a click every time.": "Un clic de moins à chaque fois.",
    "Open with Windows": "Lancer avec Windows",
    "Starts quietly in the background when you sign in.": "Démarre discrètement en arrière-plan à l'ouverture de session.",
    "Move the original to the Recycle Bin": "Envoyer l'original à la Corbeille",
    "After subtitles are added and the new file checks out. Saves disk space.": "Une fois les sous-titres ajoutés et le nouveau fichier vérifié. Libère de l'espace disque.",
    "Preferred subtitle language": "Langue de sous-titres préférée",
    "Picked first when a movie has several.": "Choisie en premier quand un film en a plusieurs.",
    "App language": "Langue de l'app",
    "Changes the app's text. Movie titles stay as they are.": "Change les textes de l'app. Les titres des films restent tels quels.",
    "Automatic (Windows)": "Automatique (Windows)",
    "This product uses the TMDB API but is not endorsed or certified by TMDB.": "Ce produit utilise l'API TMDB mais n'est ni approuvé ni certifié par TMDB.",
    "This is what you'll pick on the TV.": "C'est ce nom que vous choisirez sur la TV.",

    // ---- Messages from the app (Go side)
    "Something went wrong": "Un problème est survenu",
    "cancelled": "cancelled",
    "FFmpeg is not installed": "FFmpeg n'est pas installé",
    "FFmpeg is not installed yet": "FFmpeg n'est pas encore installé",
    "rclone is not installed yet": "rclone n'est pas encore installé",
    "the movie or subtitle file is gone": "le film ou le fichier de sous-titres a disparu",
    "that subtitle file is gone": "ce fichier de sous-titres a disparu",
    "another movie is already being prepared": "un autre film est déjà en préparation",
    "the new file has no subtitle track; the original is untouched": "le nouveau fichier n'a pas de piste de sous-titres ; l'original n'a pas été modifié",
    "could not take a frame from this movie": "impossible d'extraire une image de ce film",
    "couldn't read this movie's details": "impossible de lire les détails de ce film",
    "that folder doesn't exist any more": "ce dossier n'existe plus",
    "give your PC a name, so you can find it on the TV": "donnez un nom à votre PC pour le retrouver sur la TV",
    "the name can't be empty": "le nom ne peut pas être vide",
    "the share could not start": "le partage n'a pas pu démarrer",
    "the network port is already in use (is another copy of the share running?)": "le port réseau est déjà utilisé (une autre copie du partage tourne-t-elle ?)",
    "Windows blocked the share. Check that the firewall allows it": "Windows a bloqué le partage. Vérifiez que le pare-feu l'autorise",
    "couldn't reach TMDB (are you online?)": "impossible de joindre TMDB (êtes-vous connecté ?)",
    "TMDB didn't accept the key": "TMDB n'a pas accepté la clé",
    "no TMDB key": "pas de clé TMDB",
    "moving to Recycle Bin was cancelled": "l'envoi à la Corbeille a été annulé",
  };

  // Strings with a changing part: [pattern, French].
  const FR_PATTERNS = [
    [/^(\d+) movies?$/, (m, n) => `${n} film${n > 1 ? "s" : ""}`],
    [/^(\d+) movies?, (\d+) can get subtitles$/, (m, n, k) => `${n} film${n > 1 ? "s" : ""}, ${k} à sous-titrer`],
    [/^(\d+) movies? found$/, (m, n) => `${n} film${n > 1 ? "s" : ""} trouvé${n > 1 ? "s" : ""}`],
    [/^(\d+(?:\.\d)?) GB$/, (m, n) => `${n.replace(".", ",")} Go`],
    [/^(\d+) MB$/, (m, n) => `${n} Mo`],
    [/^On · TV sees “(.+)”$/, (m, n) => `Activé · la TV voit « ${n} »`],
    [/^TV Share (\S+) · free and open source$/, (m, v) => `TV Share ${v} · gratuit et open source`],
    [/^Rated (\d+\.\d)$/, (m, r) => `Note ${r.replace(".", ",")}`],
    [/^TMDB rating from (\d+) votes$/, (m, n) => `Note TMDB sur ${n} votes`],
    [/^Languages: (.+)$/, (m, l) => `Langues : ${langList(l)}`],
    [/^Downloading (\S+)… (\d+)%$/, (m, w, p) => `Téléchargement de ${w}… ${p} %`],
    [/^Unpacking (\S+)…$/, (m, w) => `Décompression de ${w}…`],
    [/^In VLC, go to Browse, then Local network, then “(.+)”\.$/, (m, n) => `Dans VLC, allez dans Navigation, puis Réseau local, puis « ${n} ».`],
    [/^Subtitles added to “(.+)”\. The original is in the Recycle Bin\.$/, (m, t) => `Sous-titres ajoutés à « ${t} ». L'original est dans la Corbeille.`],
    [/^Subtitles added to “(.+)”\.$/, (m, t) => `Sous-titres ajoutés à « ${t} ».`],
    [/^Add (.+) subtitles$/, (m, l) => `Ajouter les sous-titres (${langList(l)})`],
    [/^Subtitle language for (.+)$/, (m, t) => `Langue des sous-titres pour ${t}`],
    [/^(.+), (.+), (subtitles built in|subtitles can be added|no subtitles)\. Press Enter for details\.$/,
      (m, t, s, st) => `${t}, ${tr(s)}, ${{ "subtitles built in": "sous-titres intégrés", "subtitles can be added": "sous-titres à ajouter", "no subtitles": "sans sous-titres" }[st]}. Entrée pour les détails.`],
    [/^Couldn't start sharing: (.+)$/, (m, e) => `Impossible de lancer le partage : ${tr(e)}`],
    [/^Couldn't restart sharing: (.+)$/, (m, e) => `Impossible de relancer le partage : ${tr(e)}`],
    [/^Sharing stopped: (.+)$/, (m, e) => `Partage arrêté : ${tr(e)}`],
    [/^Couldn't get the helpers: (.+?)\. Check your internet connection and try again$/, (m, e) => `Impossible d'obtenir les outils : ${tr(e)}. Vérifiez votre connexion internet et réessayez`],
    [/^(rclone|FFmpeg): (.+)$/, (m, w, e) => `${w} : ${tr(e)}`],
    [/^the downloaded file doesn't match its official checksum, so it wasn't used\. Try again later$/, () => "le fichier téléchargé ne correspond pas à son empreinte officielle, il n'a donc pas été utilisé. Réessayez plus tard"],
    [/^couldn't reach (\S+) \(are you online\?\)$/, (m, s) => `impossible de joindre ${s} (êtes-vous connecté ?)`],
    [/^the download was interrupted, try again$/, () => "le téléchargement a été interrompu, réessayez"],
    [/^the new file is (\d+) s long instead of (\d+) s; the original is untouched$/, (m, a, b) => `le nouveau fichier dure ${a} s au lieu de ${b} s ; l'original n'a pas été modifié`],
    [/^FFmpeg could not build the file: (.+)$/, (m, e) => `FFmpeg n'a pas pu créer le fichier : ${e}`],
    [/^port (\d+) is already in use, maybe the share is already running in another window$/, (m, p) => `le port ${p} est déjà utilisé : le partage tourne peut-être déjà dans une autre fenêtre`],
    [/^the folder "(.+)" does not exist$/, (m, f) => `le dossier « ${f} » n'existe pas`],
    [/^(?:\w+ )?(?:mono|stereo|5\.1 surround|7\.1 surround|\d+ ch)(?:, (?:\w+ )?(?:mono|stereo|5\.1 surround|7\.1 surround|\d+ ch))*$/i, (m) => m.split(", ").map(audioFR).join(", ")],
    // Subtitle labels: "English", "English · Forced", "French · for the hard of hearing"
    [/^(\w+)((?: · (?!for the hard of hearing$)[^·]+?)?)( · for the hard of hearing)?$/,
      (m, l, v, sdh) => (LANG_FR[l] ? LANG_FR[l] + (v || "") + (sdh ? " · pour sourds et malentendants" : "") : null)],
  ];

  // Rich text blocks (with bold words): matched on the element's HTML.
  const FR_HTML = {
    "Movie night,<br>on your TV.": "Soirée film,<br>sur votre TV.",
    "One-time setup: TV Share needs two free, open-source helpers. It gets them from their official sites (about 150 MB). Allow a few minutes on a slow connection.":
      "Configuration unique : TV Share a besoin de deux outils gratuits et open source. Il les télécharge depuis leurs sites officiels (environ 150 Mo). Comptez quelques minutes avec une connexion lente.",
    "Or get \"App Installer\" from the Microsoft Store, then try again": "Ou installez « Programme d'installation d'application » depuis le Microsoft Store, puis réessayez",
    "Where are your movies?": "Où sont vos films ?",
    "Your TV will see this folder and everything inside it. Nothing else on your PC.": "Votre TV verra ce dossier et tout ce qu'il contient. Rien d'autre sur votre PC.",
    "<strong>Windows treats this Wi-Fi as public.</strong> If your TV can't find this PC, set it to <em>Private</em> (only on your home network).":
      "<strong>Windows considère ce Wi-Fi comme public.</strong> Si votre TV ne trouve pas ce PC, passez-le en <em>Privé</em> (seulement sur votre réseau domestique).",
    "<strong>New: official movie posters.</strong> TV Share can look your movies up on TMDB for posters, plot and rating. Only titles and years are sent.":
      "<strong>Nouveau : affiches officielles.</strong> TV Share peut chercher vos films sur TMDB pour les affiches, le résumé et la note. Seuls les titres et années sont envoyés.",
    "Put movies in <span id=\"emptyFolder\" class=\"path\"></span>, or drop a movie folder onto this window.":
      "Mettez des films dans <span id=\"emptyFolder\" class=\"path\"></span>, ou déposez un dossier de film sur cette fenêtre.",
    "Turn on <strong>Share with TV</strong> (top right) and keep this app open, with the PC awake, on the <strong>same Wi-Fi</strong> as the TV.":
      "Activez <strong>Partager avec la TV</strong> (en haut à droite) et gardez l'app ouverte, le PC allumé, sur le <strong>même Wi-Fi</strong> que la TV.",
    "On the TV, open <strong>VLC</strong>. It's free in the TV's app store.": "Sur la TV, ouvrez <strong>VLC</strong>. Il est gratuit dans la boutique d'apps de la TV.",
    "Go to <strong>Browse → Local network → <span class=\"tv-name\">your PC</span></strong> and pick a movie.":
      "Allez dans <strong>Navigation → Réseau local → <span class=\"tv-name\">votre PC</span></strong> et choisissez un film.",
    "<strong>Movie won't start, or VLC jumps to another file?</strong> VLC is resuming it from the very end. Long-press the movie in VLC and choose <strong>Play from start</strong>.":
      "<strong>Le film ne démarre pas, ou VLC passe à un autre fichier ?</strong> VLC le reprend à la toute fin. Faites un appui long sur le film dans VLC et choisissez <strong>Lire depuis le début</strong>.",
    "Rename your PC or change the movie folder in <button id=\"drawerSettings\" class=\"link-inline\">Settings</button>.":
      "Renommez votre PC ou changez le dossier des films dans les <button id=\"drawerSettings\" class=\"link-inline\">Paramètres</button>.",
    "Also works with Kodi, game consoles and most smart-TV media players. The first time, if Windows asks whether to allow network access, click <strong>Allow</strong>.":
      "Fonctionne aussi avec Kodi, les consoles de jeux et la plupart des lecteurs multimédias des TV. La première fois, si Windows demande d'autoriser l'accès au réseau, cliquez sur <strong>Autoriser</strong>.",
    "The first time, Windows asks whether <strong>Rclone</strong> may use the network. Rclone is the helper TV Share uses to show your movies to the TV.":
      "La première fois, Windows demande si <strong>Rclone</strong> peut utiliser le réseau. Rclone est l'outil qui permet à TV Share de montrer vos films à la TV.",
    "Click <strong>Allow</strong> (in French: <strong>Autoriser</strong>). Otherwise your TV can't see this PC.":
      "Cliquez sur <strong>Autoriser</strong>. Sinon, votre TV ne pourra pas voir ce PC.",
  };

  const LANG_FR = {
    English: "Anglais", French: "Français", Spanish: "Espagnol", German: "Allemand", Italian: "Italien",
    Portuguese: "Portugais", Arabic: "Arabe", Dutch: "Néerlandais", Turkish: "Turc", Polish: "Polonais",
    Russian: "Russe", Ukrainian: "Ukrainien", Romanian: "Roumain", Greek: "Grec", Swedish: "Suédois",
    Norwegian: "Norvégien", Danish: "Danois", Finnish: "Finnois", Czech: "Tchèque", Slovak: "Slovaque",
    Slovenian: "Slovène", Croatian: "Croate", Serbian: "Serbe", Bulgarian: "Bulgare", Hungarian: "Hongrois",
    Estonian: "Estonien", Latvian: "Letton", Lithuanian: "Lituanien", Icelandic: "Islandais", Hebrew: "Hébreu",
    Hindi: "Hindi", Indonesian: "Indonésien", Malay: "Malais", Thai: "Thaï", Vietnamese: "Vietnamien",
    Chinese: "Chinois", Japanese: "Japonais", Korean: "Coréen", Persian: "Persan",
  };
  const AUDIO_FR = { mono: "mono", stereo: "stéréo", "5.1 surround": "5.1 surround", "7.1 surround": "7.1 surround" };
  function audioFR(a) {
    const m = a.match(/^(?:(\w+) )?(mono|stereo|5\.1 surround|7\.1 surround|\d+ ch)$/i);
    if (!m) return a;
    const what = AUDIO_FR[m[2].toLowerCase()] || m[2];
    return m[1] ? `${LANG_FR[m[1]] || m[1]} ${what}` : what.charAt(0).toUpperCase() + what.slice(1);
  }
  const langList = (s) => s.split(", ").map((x) => LANG_FR[x] || x).join(", ");

  const LANGS = { fr: { words: FR, patterns: FR_PATTERNS, html: FR_HTML } };

  // Which language: the one chosen in Settings, else Windows' own.
  let choice = "auto";
  try { choice = localStorage.getItem("lang") || "auto"; } catch {}
  const sys = (navigator.language || "en").slice(0, 2).toLowerCase();
  const lang = choice === "auto" ? (LANGS[sys] ? sys : "en") : (LANGS[choice] ? choice : "en");
  const L = LANGS[lang];

  const norm = (s) => s.replace(/\s+/g, " ").trim();
  function tr(s) {
    if (!L || typeof s !== "string") return s;
    const key = norm(s);
    if (!key) return s;
    if (Object.prototype.hasOwnProperty.call(L.words, key)) {
      const lead = s.match(/^\s*/)[0], tail = s.match(/\s*$/)[0];
      return lead + L.words[key] + tail;
    }
    for (const [re, fn] of L.patterns) {
      const m = key.match(re);
      if (m) {
        const out = fn(...m);
        if (out != null) return out;
      }
    }
    return s;
  }

  const ATTRS = ["title", "aria-label", "placeholder", "alt"];
  const SKIP = new Set(["SCRIPT", "STYLE", "svg", "use", "symbol", "path", "TEXTAREA"]);
  let busy = false;

  function translateTree(root) {
    if (!L || !root) return;
    busy = true;
    try {
      const els = root.nodeType === 1 ? [root, ...root.querySelectorAll("*")] : [];
      for (const el of els) {
        if (SKIP.has(el.tagName) || el.closest("[data-no-tr]")) continue;
        const html = norm(el.innerHTML);
        if (L.html[html] !== undefined) {
          const ids = {};
          el.querySelectorAll("[id]").forEach((n) => (ids[n.id] = n.textContent));
          el.innerHTML = L.html[html];
          for (const [id, text] of Object.entries(ids)) {
            const n = el.querySelector("#" + CSS.escape(id));
            if (n && text) n.textContent = text;
          }
          continue;
        }
        for (const a of ATTRS) {
          if (el.hasAttribute(a)) {
            const v = el.getAttribute(a), t = tr(v);
            if (t !== v) el.setAttribute(a, t);
          }
        }
      }
      const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
      const nodes = [];
      if (root.nodeType === 3) nodes.push(root);
      while (walker.nextNode()) nodes.push(walker.currentNode);
      for (const n of nodes) {
        const p = n.parentElement;
        if (!p || SKIP.has(p.tagName) || p.closest("[data-no-tr]")) continue;
        const t = tr(n.nodeValue);
        if (t !== n.nodeValue) n.nodeValue = t;
      }
    } finally {
      busy = false;
    }
  }

  window.i18n = { lang, choice, t: tr, set(c) { try { localStorage.setItem("lang", c); } catch {} location.reload(); } };

  if (!L) return;
  document.documentElement.lang = lang;
  const start = () => {
    translateTree(document.body);
    new MutationObserver((muts) => {
      if (busy) return;
      for (const m of muts) {
        if (m.type === "characterData") translateTree(m.target);
        else if (m.type === "attributes") translateTree(m.target.nodeType === 1 ? m.target : null);
        else m.addedNodes.forEach((n) => translateTree(n));
      }
    }).observe(document.body, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ATTRS });
  };
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", start); else start();
})();
