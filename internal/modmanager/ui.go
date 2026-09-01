/*
Copyright 2026 DataKnifeAI.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package modmanager

const uiHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Palworld Server Manager</title>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
  <link href="https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wght@12..96,600;12..96,700;12..96,800&family=Figtree:wght@400;500;600;700;800&display=swap" rel="stylesheet" />
  <style>
    :root {
      --sky: #5eb8d4;
      --sky-deep: #1a6f78;
      --teal: #2aa8a0;
      --teal-bright: #5ed4c8;
      --grass: #4cb85a;
      --grass-deep: #2d6a40;
      --sun: #ffd56a;
      --amber: #e8a030;
      --cream: #f6eedc;
      --sand: #fff3d6;
      --paper: #fff9ee;
      --ink: #1c2e28;
      --muted: #3d554c;
      --coral: #c44b28;
      --line: rgba(28,46,40,.14);
      --font-display: "Bricolage Grotesque", "Segoe UI", sans-serif;
      --font-body: "Figtree", "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: var(--font-body);
      font-size: 1.02rem;
      line-height: 1.5;
      background:
        radial-gradient(ellipse 80% 40% at 10% -10%, rgba(42,168,160,.16), transparent 50%),
        radial-gradient(ellipse 50% 30% at 100% 0%, rgba(232,160,48,.12), transparent 45%),
        var(--cream);
      color: var(--ink);
    }
    .hero {
      position: relative;
      min-height: 16.5rem;
      display: grid;
      align-items: end;
      overflow: hidden;
      color: #fffaf0;
      background: #3a9ab0;
    }
    .hero__media { position: absolute; inset: 0; pointer-events: none; }
    .hero__photo {
      width: 100%;
      height: 100%;
      object-fit: cover;
      object-position: 62% 42%;
    }
    .hero__veil {
      position: absolute;
      inset: 0;
      background:
        linear-gradient(105deg, rgba(28,70,72,.74) 0%, rgba(28,70,72,.42) 38%, rgba(20,50,55,.1) 100%),
        linear-gradient(180deg, rgba(20,50,55,.12) 0%, transparent 36%, rgba(18,40,42,.52) 100%);
    }
    .hero__content {
      position: relative;
      z-index: 2;
      width: min(100% - 2.3rem, 64rem);
      margin: 0 auto;
      padding: 2.6rem 0 1.7rem;
      display: flex;
      justify-content: space-between;
      align-items: flex-end;
      gap: 1rem;
    }
    .hero h1 {
      margin: 0;
      font-family: var(--font-display);
      font-size: clamp(2.2rem, 6.5vw, 3.45rem);
      font-weight: 800;
      letter-spacing: -0.03em;
      line-height: 0.96;
      color: var(--sand);
      text-shadow: 0 2px 0 rgba(18,40,42,.25), 0 8px 28px rgba(18,40,42,.45);
    }
    .wave { display: block; width: 100%; height: 3.25rem; margin-top: -1px; }
    main { max-width: 64rem; margin: 0 auto; padding: 1.15rem 1.15rem 2.4rem; }
    .trail {
      display: flex; flex-wrap: wrap; gap: 0; margin: 0 0 1.15rem;
      border-bottom: 3px solid var(--grass-deep);
    }
    .trail button {
      appearance: none; border: 0; background: transparent;
      font: 700 .95rem var(--font-display); color: var(--muted);
      padding: .6rem 1rem .5rem; cursor: pointer;
      border-bottom: 3px solid transparent; margin-bottom: -3px;
    }
    .trail button[aria-selected="true"] {
      color: var(--grass-deep);
      border-bottom-color: var(--sun);
      background: linear-gradient(180deg, transparent, rgba(255,213,106,.18));
    }
    .trail button:focus-visible { outline: 2px solid var(--sky-deep); outline-offset: 2px; }
    .trail__logout {
      margin-left: auto;
      color: var(--coral);
    }
    .trail__logout:hover { color: var(--ink); }
    .btn-logout {
      flex: 0 0 auto;
      background: rgba(255,250,240,.16);
      color: #fffaf0;
      border: 1px solid rgba(255,250,240,.5);
      white-space: nowrap;
      align-self: flex-end;
    }
    .btn-logout:hover { background: rgba(255,250,240,.3); }
    .btn-logout:focus-visible { outline: 2px solid var(--sun); outline-offset: 2px; }
    .panel { display: none; }
    .panel.active { display: block; }
    .lede { margin: 0 0 1rem; color: var(--muted); font-size: .95rem; max-width: 42rem; }
    .lede + .lede { margin-top: -.45rem; }
    .warn, .note {
      background: var(--paper);
      border: 1px solid var(--line);
      border-left: 4px solid var(--sun);
      border-radius: 0 .45rem .45rem 0;
      padding: .7rem .9rem;
      margin: 0 0 1rem;
      font-size: .9rem;
    }
    .note { border-left-color: var(--teal); }
    .group {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: .65rem;
      padding: 1.05rem 1.15rem 1.15rem;
      margin: 0 0 1rem;
    }
    .group h2 {
      margin: 0 0 .4rem;
      font-family: var(--font-display);
      font-size: 1.12rem;
      font-weight: 800;
      letter-spacing: -0.02em;
    }
    .group p { margin: 0 0 .75rem; font-size: .9rem; color: var(--muted); }
    .group--warn { border-color: rgba(232,160,48,.4); box-shadow: inset 4px 0 0 var(--amber); }
    .group--danger { border-color: rgba(196,75,40,.35); box-shadow: inset 4px 0 0 var(--coral); background: #fff6f0; }
    .stats {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(13.5rem, 1fr));
      gap: .9rem;
      margin: 0 0 1.15rem;
    }
    .stat {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: .6rem;
      padding: 1.15rem 1.2rem 1.05rem;
      min-height: 6.4rem;
      box-shadow: 0 1px 0 rgba(28,46,40,.04);
    }
    .stat b {
      display: block;
      font-family: var(--font-display);
      font-size: clamp(1.45rem, 2.4vw, 1.85rem);
      font-weight: 800;
      line-height: 1.15;
      color: var(--sky-deep);
      word-break: break-all;
      margin-bottom: .4rem;
    }
    .stat span {
      font-size: .78rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: .05em;
      color: var(--muted);
    }
    table { width: 100%; border-collapse: collapse; background: var(--paper); border: 1px solid var(--line); }
    th, td { text-align: left; padding: .5rem .7rem; border-bottom: 1px solid var(--line); font-size: .92rem; }
    th { background: #e4f3ea; font-size: .72rem; text-transform: uppercase; letter-spacing: .04em; font-family: var(--font-display); }
    .muted { color: var(--muted); font-size: .85rem; }
    .row { display: flex; flex-wrap: wrap; gap: .7rem; align-items: end; margin: .85rem 0 0; }
    .row:first-child { margin-top: 0; }
    label { display: block; font-size: .75rem; color: var(--muted); margin-bottom: .2rem; font-weight: 700; }
    input[type=text], input[type=number], input[type=file], textarea {
      font: inherit; padding: .45rem .55rem; border: 1px solid var(--line); background: #fff; border-radius: .3rem; min-width: 12rem;
    }
    textarea { width: 100%; min-height: 3.2rem; }
    button, .btn { cursor: pointer; border: 0; border-radius: .35rem; padding: .48rem .85rem; font: 700 .9rem var(--font-body); }
    .btn { background: var(--sky-deep); color: #fff; text-decoration: none; display: inline-block; }
    .btn-sun { background: var(--amber); color: #1c2e22; }
    .btn-grass { background: var(--grass-deep); color: #fff; }
    .btn-danger { background: var(--coral); color: #fff; }
    .btn-ghost { background: #fff; border: 1px solid var(--line); color: var(--ink); }
    .err { color: var(--coral); margin: .4rem 0; min-height: 1.1em; }
    .ok { color: var(--grass-deep); margin: .4rem 0; }
    code { font-size: .86em; background: rgba(42,168,160,.14); padding: .05em .3em; border-radius: .25rem; }
  </style>
</head>
<body>
  <header class="hero">
    <div class="hero__media" aria-hidden="true">
      <img
        class="hero__photo"
        src="https://dataknifeai.github.io/palworld-operator/assets/hero-world-keeper.png"
        alt=""
        width="1536"
        height="1024"
        decoding="async"
        fetchpriority="high"
      />
      <div class="hero__veil"></div>
    </div>
    <div class="hero__content">
      <h1>Server Manager</h1>
      <button type="button" class="btn-logout js-logout">Log out</button>
    </div>
  </header>
  <svg class="wave" viewBox="0 0 1440 80" preserveAspectRatio="none" aria-hidden="true">
    <path d="M0 36 C260 78 480 4 720 42 C980 82 1200 12 1440 40 L1440 80 L0 80 Z" fill="#fff3d6" />
    <path d="M0 52 C220 18 420 72 720 48 C1040 22 1260 68 1440 50 L1440 80 L0 80 Z" fill="#f6eedc" opacity="0.95" />
  </svg>
  <main>
    <nav class="trail" role="tablist">
      <button type="button" role="tab" aria-selected="true" data-tab="overview">Overview</button>
      <button type="button" role="tab" aria-selected="false" data-tab="controls">Controls</button>
      <button type="button" role="tab" aria-selected="false" data-tab="saves">Saves</button>
      <button type="button" role="tab" aria-selected="false" data-tab="mods">Mods</button>
      <button type="button" class="trail__logout js-logout">Log out</button>
    </nav>

    <section id="overview" class="panel active" role="tabpanel">
      <p class="lede">Live pulse from Palworld REST on localhost — <code>/info</code>, <code>/metrics</code>, <code>/players</code>.</p>
      <p class="lede">Official API: <a href="https://docs.palworldgame.com/api/rest-api/palwold-rest-api">palwold-rest-api</a></p>
      <div class="err" id="ov-err"></div>
      <div class="stats" id="stats"></div>
      <p class="muted" id="ov-stamp"></p>
      <table>
        <thead><tr><th>Player</th><th>Level</th><th>Ping</th><th>ID</th></tr></thead>
        <tbody id="players"></tbody>
      </table>
    </section>

    <section id="controls" class="panel" role="tabpanel">
      <div class="err" id="ctl-err"></div>
      <div class="ok" id="ctl-ok"></div>

      <div class="group">
        <h2>Announce &amp; save</h2>
        <form id="announce" class="row">
          <div style="flex:1;min-width:16rem">
            <label for="announce-msg">Broadcast to the world</label>
            <textarea id="announce-msg" required placeholder="Dinner raid in ten minutes…"></textarea>
          </div>
          <button class="btn" type="submit">Announce</button>
        </form>
        <div class="row">
          <button class="btn-grass" type="button" id="save-world">Save world (REST)</button>
        </div>
      </div>

      <div class="group group--warn">
        <h2>Restart</h2>
        <p>Players disconnect. Recreate downtime until Ready. This UI restarts too.</p>
        <div class="row">
          <button class="btn-sun" type="button" id="restart">Restart pod (Recreate)</button>
        </div>
      </div>

      <div class="group group--danger">
        <h2>Shutdown</h2>
        <p>Kicks everyone, then the game process exits. Kubernetes starts a new pod. This cannot be undone from here.</p>
        <form id="shutdown" class="row">
          <div>
            <label for="shut-wait">Wait (seconds)</label>
            <input id="shut-wait" type="number" min="0" max="600" value="30" />
          </div>
          <div style="flex:1;min-width:14rem">
            <label for="shut-msg">Message</label>
            <input id="shut-msg" type="text" placeholder="World closing — grab your Pals." />
          </div>
          <button class="btn-danger" type="submit">Shutdown…</button>
        </form>
      </div>
    </section>

    <section id="saves" class="panel" role="tabpanel">
      <p class="lede">Zip of <code>SaveGames/</code> on the game PVC. Save first so the archive matches disk.</p>
      <p class="lede">Config listings never show secret values. Downloaded INIs have passwords redacted.</p>
      <div class="err" id="sv-err"></div>
      <div class="ok" id="sv-ok"></div>

      <div class="group">
        <h2>Download</h2>
        <div class="row">
          <label><input type="checkbox" id="sv-save-first" checked /> REST save before download</label>
          <label><input type="checkbox" id="sv-cfg" /> Include Config INIs (passwords redacted)</label>
          <button class="btn" type="button" id="sv-dl">Download world zip</button>
        </div>
      </div>

      <p class="muted" id="sv-meta"></p>
      <table>
        <thead><tr><th>SaveGames</th><th>Size</th></tr></thead>
        <tbody id="sv-rows"></tbody>
      </table>
      <p class="muted" style="margin-top:.75rem">Config (names only — no password values)</p>
      <table>
        <thead><tr><th>Config/LinuxServer</th><th>Size</th></tr></thead>
        <tbody id="cfg-rows"></tbody>
      </table>

      <div class="group group--danger" style="margin-top:1.15rem">
        <h2>Replace world</h2>
        <p>Upload <strong>wipes the live SaveGames tree</strong>. Players see the new world only after a restart. Confirm twice.</p>
        <form id="sv-up" class="row">
          <div>
            <label for="sv-file">Restore archive (.zip / .tar.gz)</label>
            <input id="sv-file" type="file" accept=".zip,.tar,.tar.gz,.tgz" required />
          </div>
          <label><input type="checkbox" id="sv-up-cfg" /> Also replace Config from archive</label>
          <button class="btn-danger" type="submit">Upload &amp; replace world…</button>
        </form>
      </div>
    </section>

    <section id="mods" class="panel" role="tabpanel">
      <p class="lede">Linux PalServer does <strong>not</strong> load official Workshop or <code>PalModSettings.ini</code>.</p>
      <p class="lede">Community <code>.pak</code> files go in <code>paks/~WorkshopMods</code> and <code>paks/LogicMods</code>.</p>
      <p class="lede"><a href="https://docs.palworldgame.com/settings-and-operation/mod/">Pocketpair mods (Windows)</a> · <a href="https://yorkhost.fr/docs/en/palworld/mods-ue4ss">Yorkhost PAK / UE4SS</a></p>
      <div class="row" style="margin-top:0">
        <button class="btn-ghost" type="button" data-path="">PVC root (Mods)</button>
        <button class="btn-ghost" type="button" data-path="paks/~WorkshopMods">paks/~WorkshopMods</button>
        <button class="btn-ghost" type="button" data-path="paks/LogicMods">paks/LogicMods</button>
        <button class="btn-ghost" type="button" data-path="Workshop">Workshop</button>
      </div>
      <p class="muted" id="crumb"></p>
      <div class="err" id="mod-err"></div>
      <table>
        <thead><tr><th>Name</th><th>Size</th><th></th></tr></thead>
        <tbody id="rows"></tbody>
      </table>
      <form class="row" id="upload">
        <div>
          <label for="file">Upload into current folder</label>
          <input id="file" name="file" type="file" required />
        </div>
        <button class="btn" type="submit">Upload</button>
      </form>
    </section>
  </main>
  <script>
    const $ = (id) => document.getElementById(id);
    const opts = { credentials: "same-origin" };
    let current = "";
    let statsTimer = null;

    function show(el, msg) { if (el) el.textContent = msg || ""; }
    async function api(url, init) {
      const r = await fetch(url, Object.assign({}, opts, init));
      if (r.status === 401) { throw new Error("Unauthorized"); }
      const ct = r.headers.get("content-type") || "";
      const body = ct.includes("json") ? await r.json() : await r.text();
      if (!r.ok) {
        throw new Error((body && body.error) ? body.error : r.statusText);
      }
      return body;
    }
    function fmt(n) {
      if (n == null) return "—";
      const x = Number(n);
      if (x < 1024) return x + " B";
      const u = ["KiB","MiB","GiB"];
      let i = -1, v = x;
      do { v /= 1024; i++; } while (v >= 1024 && i < u.length-1);
      return v.toFixed(1) + " " + u[i];
    }
    function pick(obj, keys) {
      if (!obj) return undefined;
      for (const k of keys) {
        if (obj[k] != null && obj[k] !== "") return obj[k];
      }
      return undefined;
    }
    function setTab(id) {
      document.querySelectorAll(".panel").forEach((p) => p.classList.toggle("active", p.id === id));
      document.querySelectorAll(".trail button[data-tab]").forEach((b) => b.setAttribute("aria-selected", b.getAttribute("data-tab") === id ? "true" : "false"));
      if (id === "overview") refreshStats();
      if (id === "saves") loadSaves();
      if (id === "mods") list(current);
    }
    document.querySelectorAll(".trail button[data-tab]").forEach((b) => {
      b.onclick = () => setTab(b.getAttribute("data-tab"));
    });
    function logout() {
      if (logout.busy) return;
      logout.busy = true;
      const nonce = "logout:" + Date.now();
      const bogus = "Basic " + btoa(nonce);
      const goHome = () => { window.location.replace("/"); };
      const timer = setTimeout(goHome, 2000);
      const done = () => { clearTimeout(timer); goHome(); };
      try {
        const xhr = new XMLHttpRequest();
        xhr.open("GET", "/logout", true, "logout", String(Date.now()));
        xhr.setRequestHeader("Authorization", bogus);
        xhr.withCredentials = true;
        xhr.onreadystatechange = function () {
          if (xhr.readyState === 4) done();
        };
        xhr.send();
      } catch (e) {
        fetch("/logout", {
          method: "GET",
          cache: "no-store",
          credentials: "include",
          headers: { Authorization: bogus }
        }).catch(function () {}).finally(done);
      }
    }
    document.querySelectorAll(".js-logout").forEach((b) => {
      b.onclick = () => logout();
    });

    function statCard(label, value) {
      const d = document.createElement("div");
      d.className = "stat";
      const b = document.createElement("b");
      b.textContent = (value == null || value === "") ? "—" : String(value);
      const s = document.createElement("span");
      s.textContent = label;
      d.append(b, s);
      return d;
    }
    function uptimeLabel(sec) {
      if (sec == null || sec === "") return "—";
      const n = Number(sec);
      if (!Number.isFinite(n)) return String(sec);
      const h = Math.floor(n / 3600);
      const m = Math.floor((n % 3600) / 60);
      return h + "h " + m + "m";
    }
    async function refreshStats() {
      show($("ov-err"), "");
      try {
        const data = await api("/api/stats");
        const info = data.info || {};
        const metrics = data.metrics || {};
        const box = $("stats");
        box.replaceChildren(
          statCard("Version", pick(info, ["version"])),
          statCard("World GUID", pick(info, ["worldguid", "worldGuid"])),
          statCard("Server", pick(info, ["servername", "serverName"])),
          statCard("Players", (pick(metrics, ["currentplayernum", "currentPlayerNum"]) ?? "—") + " / " + (pick(metrics, ["maxplayernum", "maxPlayerNum"]) ?? "—")),
          statCard("FPS", pick(metrics, ["serverfps", "fps"])),
          statCard("Days", pick(metrics, ["days"])),
          statCard("Uptime", uptimeLabel(pick(metrics, ["uptime"]))),
          statCard("Basecamps", pick(metrics, ["basecamps", "basecampnum", "numbasecamps"]))
        );
        const rows = $("players");
        rows.replaceChildren();
        const list = (data.players && (data.players.players || data.players)) || [];
        (Array.isArray(list) ? list : []).forEach((p) => {
          const tr = document.createElement("tr");
          [pick(p, ["name"]), pick(p, ["level"]), pick(p, ["ping"]), pick(p, ["userId", "userid", "playerId", "playerId"])].forEach((v) => {
            const td = document.createElement("td");
            td.textContent = v == null ? "—" : String(v);
            tr.appendChild(td);
          });
          rows.appendChild(tr);
        });
        if (data.errors && data.errors.length) show($("ov-err"), data.errors.join(" · "));
        $("ov-stamp").textContent = "Refreshed " + new Date().toLocaleTimeString();
      } catch (e) {
        show($("ov-err"), e.message);
      }
    }

    $("announce").onsubmit = async (ev) => {
      ev.preventDefault();
      show($("ctl-err"), ""); show($("ctl-ok"), "");
      try {
        const out = await api("/api/announce", { method: "POST", headers: {"Content-Type":"application/json"}, body: JSON.stringify({ message: $("announce-msg").value }) });
        show($("ctl-ok"), out.message || "Sent.");
        $("announce-msg").value = "";
      } catch (e) { show($("ctl-err"), e.message); }
    };
    $("save-world").onclick = async () => {
      show($("ctl-err"), ""); show($("ctl-ok"), "");
      try {
        const out = await api("/api/save", { method: "POST" });
        show($("ctl-ok"), out.message || "Saved.");
      } catch (e) { show($("ctl-err"), e.message); }
    };
    $("restart").onclick = async () => {
      if (!confirm("Restart this Palworld server?\n\nPlayers disconnect. Recreate means downtime until Ready. This admin UI restarts with the pod.")) return;
      show($("ctl-err"), ""); show($("ctl-ok"), "");
      try {
        const out = await api("/api/restart", { method: "POST" });
        show($("ctl-ok"), out.message || "Restart requested.");
      } catch (e) { show($("ctl-err"), e.message); }
    };
    $("shutdown").onsubmit = async (ev) => {
      ev.preventDefault();
      const wait = Number($("shut-wait").value || 0);
      if (!confirm("Shutdown the Palworld process in " + wait + "s?\n\nEveryone is kicked. The game exits, then Kubernetes starts a new pod. This cannot be undone from here.")) return;
      show($("ctl-err"), ""); show($("ctl-ok"), "");
      try {
        const out = await api("/api/shutdown", { method: "POST", headers: {"Content-Type":"application/json"}, body: JSON.stringify({ waittime: wait, message: $("shut-msg").value }) });
        show($("ctl-ok"), out.message || "Shutdown requested.");
      } catch (e) { show($("ctl-err"), e.message); }
    };

    function fillSaveRows(tbody, entries) {
      tbody.replaceChildren();
      (entries || []).forEach((e) => {
        const tr = document.createElement("tr");
        const n = document.createElement("td");
        n.textContent = e.dir ? e.name + "/" : e.name;
        const s = document.createElement("td");
        s.className = "muted";
        s.textContent = e.dir ? fmt(e.size) : fmt(e.size);
        tr.append(n, s);
        tbody.appendChild(tr);
      });
    }
    async function loadSaves() {
      show($("sv-err"), "");
      try {
        const data = await api("/api/saves");
        fillSaveRows($("sv-rows"), data.saveGames);
        fillSaveRows($("cfg-rows"), data.config);
        $("sv-meta").textContent = (data.warning || ("SaveGames " + fmt(data.totalBytes))) + (data.saveGamesRel ? " · " + data.saveGamesRel : "");
      } catch (e) { show($("sv-err"), e.message); }
    }
    $("sv-dl").onclick = async () => {
      show($("sv-err"), ""); show($("sv-ok"), "");
      try {
        if ($("sv-save-first").checked) {
          await api("/api/save", { method: "POST" });
        }
        const q = $("sv-cfg").checked ? "?includeConfig=1" : "";
        const r = await fetch("/api/saves/download" + q, opts);
        if (!r.ok) {
          const body = await r.json().catch(() => ({}));
          throw new Error(body.error || r.statusText);
        }
        const blob = await r.blob();
        const a = document.createElement("a");
        a.href = URL.createObjectURL(blob);
        a.download = "palworld-save.zip";
        a.click();
        URL.revokeObjectURL(a.href);
        show($("sv-ok"), "Download started.");
      } catch (e) { show($("sv-err"), e.message); }
    };
    $("sv-up").onsubmit = async (ev) => {
      ev.preventDefault();
      const f = $("sv-file").files[0];
      if (!f) return;
      if (!confirm("Replace the LIVE world with " + f.name + "?\n\nThis wipes the current SaveGames tree. Players see the uploaded world only after restart.")) return;
      if (!confirm("Last chance: restore " + f.name + " over the current world?")) return;
      show($("sv-err"), ""); show($("sv-ok"), "");
      const fd = new FormData();
      fd.append("file", f);
      if ($("sv-up-cfg").checked) fd.append("includeConfig", "1");
      try {
        const out = await api("/api/saves/upload", { method: "POST", body: fd });
        show($("sv-ok"), out.message || "Restored.");
        $("sv-file").value = "";
        loadSaves();
      } catch (e) { show($("sv-err"), e.message); }
    };

    function joinPath(dir, name) {
      if (!dir) return name;
      return dir.replace(/\/$/, "") + "/" + name;
    }
    async function list(path) {
      current = path || "";
      $("crumb").textContent = "Path: /" + (current || "");
      show($("mod-err"), "");
      try {
        const data = await api("/api/files?path=" + encodeURIComponent(current));
        const rows = $("rows");
        rows.replaceChildren();
        if (current) {
          const tr = document.createElement("tr");
          const td = document.createElement("td");
          td.colSpan = 3;
          const a = document.createElement("button");
          a.className = "btn-ghost";
          a.textContent = ".. (parent)";
          a.onclick = () => {
            const i = current.lastIndexOf("/");
            list(i < 0 ? "" : current.slice(0, i));
          };
          td.appendChild(a);
          tr.appendChild(td);
          rows.appendChild(tr);
        }
        (data.entries || []).forEach((e) => {
          const tr = document.createElement("tr");
          const nameTd = document.createElement("td");
          if (e.dir) {
            const b = document.createElement("button");
            b.className = "btn-ghost";
            b.textContent = e.name + "/";
            b.onclick = () => list(e.path);
            nameTd.appendChild(b);
          } else {
            const a = document.createElement("a");
            a.href = "/api/download?path=" + encodeURIComponent(e.path);
            a.textContent = e.name;
            nameTd.appendChild(a);
          }
          const sizeTd = document.createElement("td");
          sizeTd.className = "muted";
          sizeTd.textContent = e.dir ? "dir" : e.size;
          const act = document.createElement("td");
          const del = document.createElement("button");
          del.className = "btn-danger";
          del.textContent = "Delete";
          del.onclick = async () => {
            if (!confirm("Delete " + e.path + "? This cannot be undone.")) return;
            await api("/api/files?path=" + encodeURIComponent(e.path), { method: "DELETE" });
            list(current);
          };
          act.appendChild(del);
          tr.append(nameTd, sizeTd, act);
          rows.appendChild(tr);
        });
      } catch (e) { show($("mod-err"), e.message); }
    }
    document.querySelectorAll("button[data-path]").forEach((b) => {
      b.onclick = () => list(b.getAttribute("data-path") || "");
    });
    $("upload").onsubmit = async (ev) => {
      ev.preventDefault();
      const f = $("file").files[0];
      if (!f) return;
      const fd = new FormData();
      fd.append("path", current);
      fd.append("file", f);
      try {
        await api("/api/upload", { method: "POST", body: fd });
        $("file").value = "";
        list(current);
      } catch (e) { show($("mod-err"), e.message); }
    };

    refreshStats();
    statsTimer = setInterval(() => {
      if (document.getElementById("overview").classList.contains("active") && !document.hidden) refreshStats();
    }, 8000);
  </script>
</body>
</html>
`
