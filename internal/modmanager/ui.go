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
    main { max-width: 64rem; margin: 0 auto; padding: 1.15rem 1.15rem 4.2rem; }
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
    .mod-notes {
      margin: 1.35rem 0 0;
      background: var(--sand);
      border: 1px solid rgba(232,160,48,.45);
      border-radius: .65rem;
      box-shadow: inset 5px 0 0 var(--amber);
      padding: 1rem 1.15rem 1.05rem 1.25rem;
    }
    .mod-notes h2 {
      margin: 0 0 .65rem;
      font-family: var(--font-display);
      font-size: 1.05rem;
      font-weight: 800;
      letter-spacing: -0.02em;
      color: var(--ink);
    }
    .mod-notes ul {
      margin: 0;
      padding: 0;
      list-style: none;
    }
    .mod-notes li {
      position: relative;
      padding: .55rem .7rem .55rem 1.15rem;
      margin: 0 0 .45rem;
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: .4rem;
      font-size: .9rem;
      color: var(--ink);
    }
    .mod-notes li:last-child { margin-bottom: 0; }
    .mod-notes li::before {
      content: "";
      position: absolute;
      left: .45rem;
      top: .75rem;
      width: .35rem;
      height: .35rem;
      border-radius: 50%;
      background: var(--amber);
    }
    .space-meter {
      background: var(--paper);
      border: 1px solid var(--line);
      border-radius: .65rem;
      padding: .85rem 1.05rem 1rem;
      margin: 0 0 1rem;
      max-width: 36rem;
    }
    .space-meter__head {
      display: flex;
      justify-content: space-between;
      gap: .75rem;
      align-items: baseline;
      flex-wrap: wrap;
      margin-bottom: .45rem;
    }
    .space-meter__head strong {
      font-family: var(--font-display);
      font-size: .95rem;
    }
    .space-meter__head span { font-size: .82rem; color: var(--muted); }
    .space-meter__track {
      height: .7rem;
      background: rgba(28,46,40,.08);
      border: 1px solid var(--line);
      border-radius: .45rem;
      overflow: hidden;
    }
    .space-meter__bar {
      height: 100%;
      width: 0;
      background: linear-gradient(90deg, var(--teal), var(--grass));
      border-radius: .45rem;
    }
    .space-meter__hint { margin: .45rem 0 0; font-size: .82rem; color: var(--muted); }
    .btn-ghost.is-current {
      border-color: var(--teal);
      background: rgba(42,168,160,.12);
      color: var(--sky-deep);
    }
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
    .stats--image {
      grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    }
    .stats--image .stat { min-width: 0; }
    .stats--image .stat b {
      font-size: 1.02rem;
      word-break: normal;
      overflow-wrap: normal;
      white-space: nowrap;
    }
    @media (max-width: 36rem) {
      .stats--image { grid-template-columns: 1fr; }
      .stats--image .stat b {
        overflow: hidden;
        text-overflow: ellipsis;
      }
    }
    .fields {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: .85rem 1rem;
    }
    .field--wide { grid-column: 1 / -1; }
    .field input[type=text], .field textarea {
      width: 100%;
      min-width: 0;
    }
    .field.is-invalid input, .field.is-invalid textarea {
      border-color: var(--coral);
    }
    .field-err, .field-warn {
      margin: .22rem 0 0;
      font-size: .78rem;
      color: var(--coral);
    }
    .field-warn { color: #8a5a00; }
    .field-err[hidden], .field-warn[hidden] { display: none; }
    .settings-split {
      display: grid;
      grid-template-columns: 1fr minmax(15rem, 17.5rem);
      gap: 1rem 1.15rem;
      align-items: start;
    }
    .settings-key {
      background: var(--sand);
      border: 1px solid rgba(232,160,48,.4);
      border-radius: .5rem;
      padding: .7rem .8rem .65rem;
    }
    .settings-key h3 {
      margin: 0 0 .45rem;
      font-family: var(--font-display);
      font-size: .82rem;
      font-weight: 800;
    }
    .settings-key ul { margin: 0; padding: 0; list-style: none; }
    .settings-key li {
      margin: 0 0 .4rem;
      font-size: .78rem;
      line-height: 1.35;
      color: var(--ink);
    }
    .settings-key li:last-child { margin-bottom: 0; }
    .settings-key li strong { font-weight: 700; }
    @media (max-width: 48rem) {
      .settings-split { grid-template-columns: 1fr; }
    }
    .check {
      display: flex;
      align-items: center;
      gap: .45rem;
      font-size: .9rem;
      color: var(--ink);
      font-weight: 700;
      margin-bottom: 0;
    }
    .check input { width: 1.05rem; height: 1.05rem; accent-color: var(--sky-deep); }
    .badge {
      display: inline-block;
      font: 800 .72rem var(--font-display);
      text-transform: uppercase;
      letter-spacing: .06em;
      padding: .22rem .55rem;
      border-radius: .3rem;
    }
    .badge--ok { background: #e4f3ea; color: var(--grass-deep); }
    .badge--warn { background: #fff3d6; color: #7a4e00; }
    .badge--off { background: rgba(28,46,40,.08); color: var(--muted); }
    .badge--danger { background: #fde8e2; color: var(--coral); }
    .force-panel { margin-top: .9rem; }
    .force-announce {
      margin: .35rem 0 .75rem;
      background: #fff;
      border: 1px solid var(--line);
      border-radius: .4rem;
      padding: .65rem .8rem;
      font-size: .92rem;
    }
    .force-timer {
      display: flex;
      align-items: baseline;
      gap: .55rem;
      margin: 0 0 .55rem;
    }
    .force-timer strong {
      font-family: var(--font-display);
      font-size: 2.15rem;
      font-weight: 800;
      line-height: 1;
      color: var(--coral);
    }
    .force-timer span { font-size: .85rem; color: var(--muted); font-weight: 700; }
    .force-bar-track {
      height: .55rem;
      background: rgba(28,46,40,.1);
      border: 1px solid var(--line);
      border-radius: .4rem;
      overflow: hidden;
    }
    .force-bar {
      height: 100%;
      width: 0;
      background: linear-gradient(90deg, var(--amber), var(--coral));
      border-radius: .4rem;
      transition: width .2s linear;
    }
    .status-head {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 1rem;
      flex-wrap: wrap;
      margin-bottom: .75rem;
    }
    .status-head h2 { margin-bottom: 0; }
    .unsaved { font-size: .82rem; color: var(--amber); font-weight: 700; }
    @media (max-width: 40rem) {
      .fields { grid-template-columns: 1fr; }
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
    .upload-status { margin: .75rem 0 0; max-width: 28rem; }
    .upload-status[hidden] { display: none; }
    .upload-status__track {
      height: .55rem;
      background: rgba(28,46,40,.1);
      border: 1px solid var(--line);
      border-radius: .4rem;
      overflow: hidden;
    }
    .upload-status__bar {
      height: 100%;
      width: 0;
      background: linear-gradient(90deg, var(--teal), var(--grass));
      border-radius: .4rem;
      transition: width .12s ease-out;
    }
    .upload-status__bar.is-indeterminate {
      width: 32%;
      animation: upload-slide 1.05s ease-in-out infinite;
    }
    @keyframes upload-slide {
      0% { transform: translateX(-120%); }
      100% { transform: translateX(380%); }
    }
    code { font-size: .86em; background: rgba(42,168,160,.14); padding: .05em .3em; border-radius: .25rem; }
    .lede a { color: var(--sky-deep); font-weight: 700; }
    .btn-sm { padding: .28rem .55rem; font-size: .78rem; }
    .cell-actions { display: flex; flex-wrap: wrap; gap: .35rem; white-space: nowrap; }
    .chain {
      display: flex;
      flex-wrap: wrap;
      gap: .4rem;
      margin: 0 0 .65rem;
    }
    .chain-step {
      font: 800 .7rem var(--font-display);
      text-transform: uppercase;
      letter-spacing: .05em;
      padding: .28rem .55rem;
      border-radius: .3rem;
      background: rgba(28,46,40,.08);
      color: var(--muted);
    }
    .chain-step.is-active { background: #fff3d6; color: #7a4e00; }
    .chain-step.is-done { background: #e4f3ea; color: var(--grass-deep); }
    .opt-head, .opt-row {
      display: grid;
      grid-template-columns: minmax(11rem, 1.35fr) minmax(5.8rem, .7fr) minmax(9rem, 1fr);
      gap: .4rem .75rem;
      align-items: center;
    }
    .opt-head {
      margin: 0 0 .35rem;
      padding-bottom: .35rem;
      border-bottom: 1px solid var(--line);
    }
    .opt-head span {
      font-size: .72rem;
      text-transform: uppercase;
      letter-spacing: .04em;
      font-family: var(--font-display);
      font-weight: 800;
      color: var(--muted);
    }
    .opt-group {
      margin: .95rem 0 .4rem;
      font-family: var(--font-display);
      font-size: .82rem;
      font-weight: 800;
      letter-spacing: -0.02em;
    }
    .opt-row { margin: 0 0 .4rem; }
    .opt-row label { margin: 0; }
    .opt-live {
      font-size: .88rem;
      font-weight: 700;
      color: var(--sky-deep);
    }
    .opt-row input, .opt-row select {
      width: 100%;
      min-width: 0;
      font: inherit;
      padding: .4rem .5rem;
      border: 1px solid var(--line);
      background: #fff;
      border-radius: .3rem;
    }
    .opt-row.is-invalid input, .opt-row.is-invalid select { border-color: var(--coral); }
    .platforms { display: flex; flex-wrap: wrap; gap: .65rem 1rem; }
    .cred-row {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: .55rem .75rem;
      margin: 0 0 .65rem;
    }
    .cred-row:last-child { margin-bottom: 0; }
    .cred-label {
      min-width: 7.5rem;
      font-size: .75rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: .04em;
      color: var(--muted);
    }
    .cred-mask {
      font-family: ui-monospace, monospace;
      letter-spacing: .14em;
      color: var(--muted);
    }
    .cred-once {
      font-family: ui-monospace, monospace;
      font-size: .88rem;
      background: rgba(42,168,160,.14);
      padding: .15rem .4rem;
      border-radius: .25rem;
    }
    @media (max-width: 40rem) {
      .opt-head { display: none; }
      .opt-row { grid-template-columns: 1fr; }
    }
    .gh-fab {
      position: fixed;
      right: 1.1rem;
      bottom: 1.1rem;
      z-index: 20;
      width: 2.4rem;
      height: 2.4rem;
      display: grid;
      place-items: center;
      border-radius: 50%;
      background: var(--paper);
      color: var(--sky-deep);
      border: 1px solid var(--line);
      box-shadow: 0 3px 12px rgba(28, 46, 40, .14);
      text-decoration: none;
    }
    .gh-fab:hover {
      background: var(--teal);
      color: #fffaf0;
      border-color: var(--teal);
    }
    .gh-fab:focus-visible {
      outline: 2px solid var(--sky-deep);
      outline-offset: 3px;
    }
    .gh-fab svg { width: 1.15rem; height: 1.15rem; display: block; }
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
      <button type="button" role="tab" aria-selected="false" data-tab="updates">Updates</button>
      <button type="button" role="tab" aria-selected="false" data-tab="saves">Saves</button>
      <button type="button" role="tab" aria-selected="false" data-tab="mods">Mods</button>
      <button type="button" role="tab" aria-selected="false" data-tab="settings">Settings</button>
    </nav>

    <section id="overview" class="panel active" role="tabpanel">
      <p class="lede">Live pulse from Palworld REST on localhost — <code>/info</code>, <code>/metrics</code>, <code>/players</code>.</p>
      <p class="lede">Official API: <a href="https://docs.palworldgame.com/api/rest-api/palwold-rest-api">palwold-rest-api</a></p>
      <div class="err" id="ov-err"></div>
      <div class="ok" id="ov-ok"></div>
      <div class="stats" id="stats"></div>
      <p class="muted" id="ov-stamp"></p>
      <div class="row">
        <div style="flex:1;min-width:16rem">
          <label for="ov-msg">Message</label>
          <input id="ov-msg" type="text" placeholder="Optional" autocomplete="off" />
        </div>
      </div>
      <table>
        <thead><tr><th>Player</th><th>Level</th><th>Ping</th><th>ID</th><th></th></tr></thead>
        <tbody id="players"></tbody>
      </table>

      <div class="group" style="margin-top:1.15rem">
        <h2>Ban list</h2>
        <p class="muted" id="ban-meta"></p>
        <table>
          <thead><tr><th>ID</th><th></th></tr></thead>
          <tbody id="bans"></tbody>
        </table>
      </div>
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
          <label class="check" for="ctl-save-first">
            <input type="checkbox" id="ctl-save-first" checked />
            Save first
          </label>
        </div>
        <div class="row">
          <button class="btn-sun" type="button" id="restart">Save &amp; restart</button>
        </div>
        <div class="force-panel" id="ctl-restart-panel" hidden>
          <div class="chain" id="ctl-restart-chain" aria-live="polite">
            <span class="chain-step" data-step="save">Save</span>
            <span class="chain-step" data-step="roll">Recreate</span>
          </div>
          <p class="force-announce" id="ctl-restart-announce"></p>
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

    <section id="updates" class="panel" role="tabpanel">
      <div class="err" id="upd-err"></div>
      <div class="ok" id="upd-ok"></div>
      <div class="group">
        <div class="status-head">
          <h2>Image</h2>
          <span class="badge badge--ok" id="upd-badge">Up to date</span>
        </div>
        <div class="stats stats--image" id="upd-stats"></div>
        <div class="row">
          <button class="btn" type="button" id="upd-check">Check now</button>
          <span id="upd-force-wrap" hidden>
            <button class="btn-danger" type="button" id="upd-force">Save &amp; force update</button>
          </span>
        </div>
        <div class="force-panel" id="upd-force-panel" hidden>
          <div class="chain" id="upd-force-chain" aria-live="polite">
            <span class="chain-step" data-step="save">Save</span>
            <span class="chain-step" data-step="announce">Announce</span>
            <span class="chain-step" data-step="roll">Recreate</span>
          </div>
          <p class="force-announce" id="upd-force-announce"></p>
          <div class="force-timer" aria-live="polite">
            <strong id="upd-force-left">10</strong>
            <span id="upd-force-left-label">seconds</span>
          </div>
          <div class="force-bar-track">
            <div class="force-bar" id="upd-force-bar"></div>
          </div>
        </div>
      </div>
      <div class="group">
        <div class="status-head">
          <h2>Settings</h2>
          <span class="unsaved" id="upd-dirty" hidden>Unsaved</span>
        </div>
        <div class="settings-split">
        <form id="upd-form">
          <div class="fields">
            <div class="field field--wide">
              <label class="check" for="upd-auto">
                <input type="checkbox" id="upd-auto" />
                Auto-update
              </label>
            </div>
            <div class="field">
              <label for="upd-interval">Check interval</label>
              <input id="upd-interval" type="text" placeholder="6h" autocomplete="off" />
              <p class="field-err" id="err-interval" hidden></p>
            </div>
            <div class="field">
              <label for="upd-check-cron">Check schedule</label>
              <input id="upd-check-cron" type="text" placeholder="" autocomplete="off" />
              <p class="field-err" id="err-check-cron" hidden></p>
            </div>
            <div class="field">
              <label for="upd-apply-cron">Apply schedule</label>
              <input id="upd-apply-cron" type="text" placeholder="" autocomplete="off" />
              <p class="field-err" id="err-apply-cron" hidden></p>
            </div>
            <div class="field">
              <label for="upd-tz">Time zone</label>
              <input id="upd-tz" type="text" placeholder="UTC" autocomplete="off" />
              <p class="field-err" id="err-tz" hidden></p>
            </div>
            <div class="field field--wide">
              <label for="upd-repo">Image repository</label>
              <input id="upd-repo" type="text" placeholder="ghcr.io/pocketpairjp/palserver" autocomplete="off" />
              <p class="field-err" id="err-repo" hidden></p>
            </div>
            <div class="field field--wide">
              <label class="check" for="upd-empty">
                <input type="checkbox" id="upd-empty" checked />
                Only when empty
              </label>
            </div>
            <div class="field field--wide">
              <label class="check" for="upd-notify">
                <input type="checkbox" id="upd-notify" />
                Notify players
              </label>
            </div>
            <div class="field field--wide">
              <label for="upd-notify-sched">Notify schedule</label>
              <input id="upd-notify-sched" type="text" placeholder="60m, 30m, 15m, 5m, 1m, 30s, 10s" autocomplete="off" />
              <p class="field-err" id="err-notify-sched" hidden></p>
            </div>
            <div class="field field--wide">
              <label for="upd-notify-msg">Notify message</label>
              <textarea id="upd-notify-msg" placeholder=""></textarea>
              <p class="field-warn" id="warn-notify-msg" hidden></p>
            </div>
          </div>
          <div class="row">
            <button class="btn-grass" type="submit">Save</button>
            <button class="btn-ghost" type="button" id="upd-reset">Reset defaults</button>
          </div>
        </form>
        <aside class="settings-key" aria-label="Setting formulas">
          <h3>Defaults</h3>
          <ul>
            <li><strong>Auto-update</strong> — opt-in pin. Off</li>
            <li><strong>Check interval</strong> — poll cadence. <code>6h</code></li>
            <li><strong>Check schedule</strong> — cron instead of interval</li>
            <li><strong>Apply schedule</strong> — apply-window cron</li>
            <li><strong>Time zone</strong> — cron zone. <code>UTC</code></li>
            <li><strong>Image repository</strong> — <code>ghcr.io/pocketpairjp/palserver</code></li>
            <li><strong>Only when empty</strong> — wait for 0 players. On</li>
            <li><strong>Notify players</strong> — announce first. Off</li>
            <li><strong>Notify schedule</strong> — <code>60m, 30m, 15m, 5m, 1m, 30s, 10s</code></li>
            <li><strong>Notify message</strong> — <code>{version}</code> <code>{image}</code> <code>{remaining}</code></li>
          </ul>
        </aside>
        </div>
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
        <div class="upload-status" id="sv-progress" hidden>
          <div class="upload-status__track">
            <div class="upload-status__bar" id="sv-progress-bar"></div>
          </div>
          <p class="muted" id="sv-progress-label"></p>
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
      <p class="lede"><strong>Palworld Server does load community pak files.</strong> Drop community <code>.pak</code> files in <code>paks/~WorkshopMods</code> or <code>paks/LogicMods</code>.</p>
      <p class="lede"><a href="https://docs.palworldgame.com/settings-and-operation/mod/">Pocketpair mods</a></p>
      <div class="space-meter" aria-label="Mods PVC space">
        <div class="space-meter__head">
          <strong>Mods PVC space</strong>
          <span id="mod-space-label">Checking space…</span>
        </div>
        <div class="space-meter__track">
          <div class="space-meter__bar" id="mod-space-bar" style="width:0%"></div>
        </div>
        <p class="space-meter__hint" id="mod-space-hint">Upload is rejected if the file is larger than free space, so a mid-write fill of the PVC cannot happen.</p>
      </div>
      <div class="row" style="margin-top:0">
        <button class="btn-ghost" type="button" data-path="">PVC root (Mods)</button>
        <button class="btn-ghost" type="button" data-path="paks/~WorkshopMods">paks/~WorkshopMods</button>
        <button class="btn-ghost" type="button" data-path="paks/LogicMods">paks/LogicMods</button>
        <button class="btn-ghost" type="button" data-path="Workshop">Workshop</button>
      </div>
      <p class="muted" id="crumb"></p>
      <div class="err" id="mod-err"></div>
      <div class="ok" id="mod-ok"></div>
      <div class="upload-status" id="mod-progress" hidden>
        <div class="upload-status__track">
          <div class="upload-status__bar" id="mod-progress-bar"></div>
        </div>
        <p class="muted" id="mod-progress-label"></p>
      </div>
      <table>
        <thead><tr><th>Name</th><th>Size</th><th></th></tr></thead>
        <tbody id="rows"></tbody>
      </table>
      <form class="row" id="upload">
        <div>
          <label for="file">Upload .pak into current folder</label>
          <input id="file" name="file" type="file" accept=".pak" required />
          <p class="muted" id="mod-upload-hint" style="margin:.35rem 0 0">Only <code>.pak</code> files are accepted. A file larger than free space is rejected before upload.</p>
        </div>
        <button class="btn" type="submit">Upload</button>
      </form>
      <aside class="mod-notes" aria-labelledby="mod-notes-heading">
        <h2 id="mod-notes-heading">Notes</h2>
        <ul>
          <li>Official Workshop, <code>PalModSettings.ini</code>, <code>-workshopdir</code>, UE4SS, Lua, and Win64 DLLs are not supported.</li>
          <li>Both the client and the server must have the mod installed for it to work.</li>
          <li>Mods typically align to PC players.</li>
          <li>If Crossplay is enabled, console players may fail to connect if unsupported mods are enabled on the server.</li>
        </ul>
      </aside>
    </section>

    <section id="settings" class="panel" role="tabpanel">
      <p class="lede"><a href="https://docs.palworldgame.com/settings-and-operation/configuration/">Official settings</a></p>
      <div class="err" id="set-err"></div>
      <div class="ok" id="set-ok"></div>

      <div class="group">
        <div class="status-head">
          <h2>Server profile</h2>
          <span class="unsaved" id="prof-dirty" hidden>Unsaved</span>
        </div>
        <div class="settings-split">
          <form id="prof-form">
            <div class="fields">
              <div class="field field--wide">
                <label for="prof-name">Name</label>
                <input id="prof-name" type="text" autocomplete="off" />
              </div>
              <div class="field field--wide">
                <label for="prof-desc">Description</label>
                <textarea id="prof-desc"></textarea>
              </div>
              <div class="field">
                <label for="prof-max">Max players</label>
                <input id="prof-max" type="number" min="1" max="32" />
                <p class="field-err" id="err-prof-max" hidden></p>
              </div>
              <div class="field field--wide">
                <label>Crossplay</label>
                <div class="platforms">
                  <label class="check"><input type="checkbox" id="prof-steam" /> Steam</label>
                  <label class="check"><input type="checkbox" id="prof-xbox" /> Xbox</label>
                  <label class="check"><input type="checkbox" id="prof-ps5" /> PS5</label>
                  <label class="check"><input type="checkbox" id="prof-mac" /> Mac</label>
                </div>
              </div>
              <div class="field field--wide">
                <label class="check" for="prof-community">
                  <input type="checkbox" id="prof-community" />
                  Community listing
                </label>
              </div>
            </div>
            <div class="row">
              <button class="btn-ghost" type="button" id="prof-reset">Reset defaults</button>
            </div>
          </form>
          <aside class="settings-key" aria-label="Profile defaults">
            <h3>Defaults</h3>
            <ul>
              <li><strong>Name</strong> — empty</li>
              <li><strong>Description</strong> — empty</li>
              <li><strong>Max players</strong> — <code>4</code></li>
              <li><strong>Crossplay</strong> — <code>(Steam,Xbox,PS5,Mac)</code></li>
              <li><strong>Community listing</strong> — Off</li>
            </ul>
          </aside>
        </div>
      </div>

      <div class="group">
        <div class="status-head">
          <h2>Game settings</h2>
          <span class="unsaved" id="opt-dirty" hidden>Unsaved</span>
        </div>
        <p class="muted">Empty inherits the game default. <code>spec.optionSettings</code></p>
        <div class="opt-head">
          <span>Setting</span>
          <span>Live</span>
          <span>Desired</span>
        </div>
        <form id="opt-form"></form>
        <div class="row">
          <button class="btn-ghost" type="button" id="opt-reset">Reset defaults</button>
        </div>
        <aside class="settings-key" style="margin-top:1rem" aria-label="Game setting defaults">
          <h3>Defaults</h3>
          <ul>
            <li><strong>Empty</strong> — inherit game / operator default</li>
            <li><strong>DeathPenalty</strong> — <code>None</code> <code>Item</code> <code>ItemAndEquipment</code> <code>All</code></li>
            <li><strong>RandomizerType</strong> — <code>None</code> <code>Region</code> <code>All</code></li>
            <li><strong>Booleans</strong> — <code>True</code> / <code>False</code></li>
            <li><strong>Rates</strong> — number ≥ 0</li>
          </ul>
        </aside>
      </div>
      <div class="group group--warn">
        <h2>Apply &amp; restart</h2>
        <div class="row">
          <button class="btn-sun" type="button" id="set-apply">Apply &amp; restart</button>
        </div>
      </div>

      <div class="group group--danger">
        <h2>Credentials</h2>
        <div class="cred-row">
          <span class="cred-label">Join</span>
          <span class="badge badge--ok" id="cred-join-badge">Set</span>
          <span class="cred-mask" id="cred-join-mask">••••••••</span>
          <span class="cred-once" id="cred-join-once" hidden></span>
          <button class="btn-ghost btn-sm" type="button" id="cred-join-copy">Copy</button>
          <button class="btn-danger btn-sm" type="button" id="cred-join-rotate">Rotate &amp; restart</button>
        </div>
        <div class="cred-row">
          <span class="cred-label">Admin</span>
          <span class="badge badge--ok" id="cred-admin-badge">Set</span>
          <span class="cred-mask" id="cred-admin-mask">••••••••</span>
          <span class="cred-once" id="cred-admin-once" hidden></span>
          <button class="btn-ghost btn-sm" type="button" id="cred-admin-copy">Copy</button>
          <button class="btn-danger btn-sm" type="button" id="cred-admin-rotate">Rotate &amp; restart</button>
        </div>
      </div>
    </section>
  </main>
  <a
    class="gh-fab"
    href="https://github.com/DataKnifeAI/palworld-operator"
    title="GitHub"
    aria-label="GitHub"
  >
    <svg viewBox="0 0 16 16" aria-hidden="true">
      <path fill="currentColor" d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
    </svg>
  </a>
  <script>
    const $ = (id) => document.getElementById(id);
    const opts = { credentials: "same-origin" };
    const defaultModsPath = "paks/~WorkshopMods";
    let current = defaultModsPath;
    let modsFreeBytes = 0;
    let statsTimer = null;

    function show(el, msg) { if (el) el.textContent = msg || ""; }
    async function api(url, init) {
      const r = await fetch(url, Object.assign({}, opts, init));
      if (r.status === 401) { throw new Error("Unauthorized"); }
      const ct = r.headers.get("content-type") || "";
      const body = ct.includes("json") ? await r.json() : await r.text();
      if (!r.ok) {
        const err = new Error((body && body.error) ? body.error : r.statusText);
        if (body && body.fields) err.fields = body.fields;
        throw err;
      }
      return body;
    }
    function fmt(n) {
      if (n == null) return "—";
      const x = Number(n);
      if (x < 1024) return x + " B";
      if (x < 1024 * 1024) return (x / 1024).toFixed(1) + " KB";
      if (x < 1024 * 1024 * 1024) return (x / (1024 * 1024)).toFixed(1) + " MB";
      return (x / (1024 * 1024 * 1024)).toFixed(1) + " GB";
    }
    function isPakName(name) {
      return /\.pak$/i.test(name || "");
    }
    function renderSpace(space) {
      const used = Number(space && space.used) || 0;
      const free = Number(space && space.free) || 0;
      const total = Number(space && space.total) || 0;
      modsFreeBytes = free;
      const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0;
      const bar = $("mod-space-bar");
      const label = $("mod-space-label");
      const uploadHint = $("mod-upload-hint");
      if (bar) bar.style.width = pct + "%";
      if (label) label.textContent = fmt(used) + " used · " + fmt(free) + " free of " + fmt(total);
      if (uploadHint) uploadHint.innerHTML = "Only <code>.pak</code> files are accepted. A file larger than " + fmt(free) + " free is rejected before upload.";
    }
    async function loadSpace() {
      try {
        const space = await api("/api/space");
        renderSpace(space);
      } catch (e) {
        const label = $("mod-space-label");
        if (label) label.textContent = "Space unavailable";
      }
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
      if (id === "updates") loadUpdates();
      if (id === "saves") loadSaves();
      if (id === "mods") { list(current); loadSpace(); }
      if (id === "settings") loadSettings();
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
    function makeBtn(cls, text, fn) {
      const b = document.createElement("button");
      b.type = "button";
      b.className = cls;
      b.textContent = text;
      b.onclick = fn;
      return b;
    }
    function playerMessage() {
      return ($("ov-msg") && $("ov-msg").value.trim()) || "";
    }
    function playerId(p) {
      return pick(p, ["userId", "userid", "playerId", "playerID"]) || "";
    }
    function renderBans(bans, path) {
      const rows = $("bans");
      if (!rows) return;
      rows.replaceChildren();
      const list = Array.isArray(bans) ? bans : [];
      if (!list.length) {
        const tr = document.createElement("tr");
        const td = document.createElement("td");
        td.colSpan = 2;
        td.className = "muted";
        td.textContent = "None";
        tr.appendChild(td);
        rows.appendChild(tr);
      } else {
        list.forEach((b) => {
          const id = b.id || b.userid || "";
          const tr = document.createElement("tr");
          const idTd = document.createElement("td");
          idTd.textContent = id;
          const act = document.createElement("td");
          act.className = "cell-actions";
          act.appendChild(makeBtn("btn-ghost btn-sm", "Unban", () => unbanPlayer(id)));
          tr.append(idTd, act);
          rows.appendChild(tr);
        });
      }
      const meta = $("ban-meta");
      if (meta) meta.textContent = list.length ? ((path || "banlist.txt") + " · " + list.length) : ((path || "banlist.txt") + " · empty");
    }
    async function loadBans() {
      try {
        const data = await api("/api/bans");
        renderBans(data.bans, data.path);
      } catch (e) {
        renderBans([], "");
        if ($("ban-meta")) $("ban-meta").textContent = e.message;
      }
    }
    async function kickPlayer(id, name) {
      const msg = playerMessage();
      if (!confirm("Kick " + (name || id) + "?" + (msg ? "\n\n" + msg : ""))) return;
      show($("ov-err"), ""); show($("ov-ok"), "");
      try {
        const out = await api("/api/kick", { method: "POST", headers: {"Content-Type":"application/json"}, body: JSON.stringify({ userid: id, message: msg }) });
        show($("ov-ok"), out.message || ("Kicked " + (name || id) + "."));
        refreshStats();
      } catch (e) { show($("ov-err"), e.message); }
    }
    async function banPlayer(id, name) {
      const msg = playerMessage();
      if (!confirm("Ban " + (name || id) + "?\n\nThey cannot rejoin until Unban." + (msg ? "\n\n" + msg : ""))) return;
      show($("ov-err"), ""); show($("ov-ok"), "");
      try {
        const out = await api("/api/ban", { method: "POST", headers: {"Content-Type":"application/json"}, body: JSON.stringify({ userid: id, message: msg }) });
        show($("ov-ok"), out.message || ("Banned " + (name || id) + "."));
        refreshStats();
      } catch (e) { show($("ov-err"), e.message); }
    }
    async function unbanPlayer(id) {
      if (!confirm("Unban " + id + "?")) return;
      show($("ov-err"), ""); show($("ov-ok"), "");
      try {
        const out = await api("/api/unban", { method: "POST", headers: {"Content-Type":"application/json"}, body: JSON.stringify({ userid: id }) });
        show($("ov-ok"), out.message || "Unbanned.");
        refreshStats();
      } catch (e) { show($("ov-err"), e.message); }
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
          const id = playerId(p);
          const name = pick(p, ["name"]);
          [name, pick(p, ["level"]), pick(p, ["ping"]), id].forEach((v) => {
            const td = document.createElement("td");
            td.textContent = v == null ? "—" : String(v);
            tr.appendChild(td);
          });
          const act = document.createElement("td");
          act.className = "cell-actions";
          if (id) {
            act.append(
              makeBtn("btn-ghost btn-sm", "Kick", () => kickPlayer(id, name)),
              makeBtn("btn-danger btn-sm", "Ban", () => banPlayer(id, name))
            );
          }
          tr.appendChild(act);
          rows.appendChild(tr);
        });
        if (data.errors && data.errors.length) show($("ov-err"), data.errors.join(" · "));
        $("ov-stamp").textContent = "Refreshed " + new Date().toLocaleTimeString();
        await loadBans();
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
    function paintChain(rootId, active) {
      const order = ["save", "announce", "roll"];
      const idx = order.indexOf(active);
      document.querySelectorAll("#" + rootId + " .chain-step").forEach((el) => {
        const step = el.getAttribute("data-step");
        const si = order.indexOf(step);
        el.classList.toggle("is-active", step === active);
        el.classList.toggle("is-done", idx >= 0 && si >= 0 && si < idx);
      });
    }
    function syncRestartLabel() {
      const btn = $("restart");
      const saveFirst = $("ctl-save-first");
      if (btn && saveFirst) btn.textContent = saveFirst.checked ? "Save & restart" : "Restart pod (Recreate)";
    }
    if ($("ctl-save-first")) $("ctl-save-first").onchange = syncRestartLabel;
    let restartRunning = false;
    $("restart").onclick = async () => {
      if (restartRunning) return;
      const saveFirst = $("ctl-save-first") && $("ctl-save-first").checked;
      if (!confirm((saveFirst ? "Save & restart" : "Restart") + " this Palworld server?\n\nPlayers disconnect. Recreate means downtime until Ready. This admin UI restarts with the pod.")) return;
      show($("ctl-err"), ""); show($("ctl-ok"), "");
      restartRunning = true;
      $("restart").disabled = true;
      const panel = $("ctl-restart-panel");
      const announce = $("ctl-restart-announce");
      if (panel) panel.hidden = false;
      try {
        if (saveFirst) {
          paintChain("ctl-restart-chain", "save");
          if (announce) announce.textContent = "Saving…";
        } else {
          paintChain("ctl-restart-chain", "roll");
          if (announce) announce.textContent = "Recreate…";
        }
        const out = await api("/api/restart", { method: "POST", headers: {"Content-Type":"application/json"}, body: JSON.stringify({ saveFirst: !!saveFirst }) });
        paintChain("ctl-restart-chain", "roll");
        document.querySelectorAll("#ctl-restart-chain .chain-step").forEach((el) => {
          el.classList.remove("is-active");
          el.classList.add("is-done");
        });
        if (announce) announce.textContent = out.message || "Recreate requested.";
        show($("ctl-ok"), out.message || (saveFirst ? "Saved, then Recreate." : "Recreate requested."));
      } catch (e) { show($("ctl-err"), e.message); }
      finally {
        restartRunning = false;
        $("restart").disabled = false;
        setTimeout(() => { if (panel) panel.hidden = true; }, 1200);
      }
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
    function showSaveProgress(text, loaded, total) {
      const wrap = $("sv-progress");
      const bar = $("sv-progress-bar");
      const label = $("sv-progress-label");
      if (!wrap || !bar || !label) return;
      wrap.hidden = false;
      if (total > 0) {
        const pct = Math.min(100, Math.round((loaded / total) * 100));
        bar.classList.remove("is-indeterminate");
        bar.style.width = pct + "%";
        label.textContent = text;
        return;
      }
      bar.classList.add("is-indeterminate");
      bar.style.width = "";
      label.textContent = text;
    }
    function setDownloadProgress(loaded, total) {
      if (total > 0) {
        const pct = Math.min(100, Math.round((loaded / total) * 100));
        showSaveProgress("Downloading " + fmt(loaded) + " / " + fmt(total) + " (" + pct + "%)", loaded, total);
        return;
      }
      showSaveProgress(loaded > 0 ? ("Downloading " + fmt(loaded) + "…") : "Downloading…", loaded, 0);
    }
    function filenameFromDisposition(cd) {
      const m = /filename\*?=(?:UTF-8''|"?)([^";]+)/i.exec(cd || "");
      if (!m) return "palworld-save.zip";
      try { return decodeURIComponent(m[1].replace(/"$/, "")); } catch (e) { return m[1]; }
    }
    function downloadWithProgress(url, onProgress) {
      return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open("GET", url);
        xhr.withCredentials = true;
        xhr.responseType = "blob";
        xhr.onprogress = (ev) => {
          onProgress(ev.loaded, ev.lengthComputable ? ev.total : 0);
        };
        xhr.onload = () => {
          const fail = (msg) => reject(new Error(msg || xhr.statusText || "Download failed"));
          if (xhr.status === 401) {
            fail("Unauthorized");
            return;
          }
          if (xhr.status < 200 || xhr.status >= 300) {
            const blob = xhr.response;
            if (blob && typeof blob.text === "function") {
              blob.text().then((t) => {
                try {
                  const body = JSON.parse(t);
                  fail((body && body.error) ? body.error : (xhr.statusText || "Download failed"));
                } catch (e) {
                  fail(xhr.statusText || "Download failed");
                }
              }).catch(() => fail(xhr.statusText || "Download failed"));
              return;
            }
            fail(xhr.statusText || "Download failed");
            return;
          }
          resolve({
            blob: xhr.response,
            name: filenameFromDisposition(xhr.getResponseHeader("content-disposition"))
          });
        };
        xhr.onerror = () => reject(new Error("Network error during download"));
        xhr.onabort = () => reject(new Error("Download aborted"));
        xhr.send();
      });
    }
    $("sv-dl").onclick = async () => {
      show($("sv-err"), ""); show($("sv-ok"), "");
      const btn = $("sv-dl");
      if (btn) btn.disabled = true;
      try {
        if ($("sv-save-first").checked) {
          showSaveProgress("Saving world…", 0, 0);
          await api("/api/save", { method: "POST" });
        }
        setDownloadProgress(0, 0);
        const q = $("sv-cfg").checked ? "?includeConfig=1" : "";
        const out = await downloadWithProgress("/api/saves/download" + q, setDownloadProgress);
        const a = document.createElement("a");
        a.href = URL.createObjectURL(out.blob);
        a.download = out.name || "palworld-save.zip";
        a.click();
        URL.revokeObjectURL(a.href);
        const size = out.blob && out.blob.size ? " (" + fmt(out.blob.size) + ")" : "";
        const done = "Downloaded " + (out.name || "palworld-save.zip") + size;
        showSaveProgress(done, 1, 1);
        show($("sv-ok"), done + ".");
      } catch (e) {
        if ($("sv-progress")) $("sv-progress").hidden = true;
        show($("sv-err"), e.message);
      } finally {
        if (btn) btn.disabled = false;
      }
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
    function markPathButtons() {
      document.querySelectorAll("button[data-path]").forEach((b) => {
        b.classList.toggle("is-current", (b.getAttribute("data-path") || "") === current);
      });
    }
    async function list(path) {
      current = path || "";
      $("crumb").textContent = "Path: /" + (current || "");
      show($("mod-err"), "");
      show($("mod-ok"), "");
      markPathButtons();
      if ($("mod-progress")) $("mod-progress").hidden = true;
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
          sizeTd.textContent = e.dir ? "dir" : fmt(e.size);
          const act = document.createElement("td");
          const del = document.createElement("button");
          del.className = "btn-danger";
          del.textContent = "Delete";
          del.onclick = async () => {
            if (!confirm("Delete " + e.path + "? This cannot be undone.")) return;
            await api("/api/files?path=" + encodeURIComponent(e.path), { method: "DELETE" });
            await list(current);
            loadSpace();
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
    function setUploadProgress(loaded, total) {
      const wrap = $("mod-progress");
      const bar = $("mod-progress-bar");
      const label = $("mod-progress-label");
      if (!wrap || !bar || !label) return;
      wrap.hidden = false;
      if (total > 0) {
        const pct = Math.min(100, Math.round((loaded / total) * 100));
        bar.classList.remove("is-indeterminate");
        bar.style.width = pct + "%";
        label.textContent = "Uploading " + fmt(loaded) + " / " + fmt(total) + " (" + pct + "%)";
        return;
      }
      bar.classList.add("is-indeterminate");
      bar.style.width = "";
      label.textContent = loaded > 0 ? ("Uploading " + fmt(loaded) + "…") : "Uploading…";
    }
    function uploadWithProgress(url, fd, onProgress) {
      return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open("POST", url);
        xhr.withCredentials = true;
        if (xhr.upload) {
          xhr.upload.onprogress = (ev) => {
            onProgress(ev.loaded, ev.lengthComputable ? ev.total : 0);
          };
        }
        xhr.onload = () => {
          const ct = xhr.getResponseHeader("content-type") || "";
          let body = xhr.responseText;
          if (ct.includes("json")) {
            try { body = JSON.parse(xhr.responseText); } catch (e) { /* keep text */ }
          }
          if (xhr.status === 401) {
            reject(new Error("Unauthorized"));
            return;
          }
          if (xhr.status < 200 || xhr.status >= 300) {
            const msg = (body && body.error) ? body.error : (xhr.statusText || "Upload failed");
            reject(new Error(msg));
            return;
          }
          resolve(body);
        };
        xhr.onerror = () => reject(new Error("Network error during upload"));
        xhr.onabort = () => reject(new Error("Upload aborted"));
        xhr.send(fd);
      });
    }
    $("upload").onsubmit = async (ev) => {
      ev.preventDefault();
      const f = $("file").files[0];
      if (!f) return;
      show($("mod-err"), "");
      show($("mod-ok"), "");
      if (!isPakName(f.name)) {
        show($("mod-err"), "Only .pak files are supported.");
        return;
      }
      if (modsFreeBytes > 0 && f.size > modsFreeBytes) {
        show($("mod-err"), "File is larger than " + fmt(modsFreeBytes) + " free on the mods PVC. Upload rejected.");
        return;
      }
      const btn = ev.target.querySelector("button[type=submit]");
      setUploadProgress(0, f.size || 0);
      if (btn) btn.disabled = true;
      const fd = new FormData();
      fd.append("path", current);
      fd.append("file", f);
      try {
        const out = await uploadWithProgress("/api/upload", fd, setUploadProgress);
        $("file").value = "";
        await list(current);
        loadSpace();
        $("mod-progress").hidden = false;
        $("mod-progress-bar").classList.remove("is-indeterminate");
        $("mod-progress-bar").style.width = "100%";
        $("mod-progress-label").textContent = "Uploaded " + (out && out.name ? out.name : f.name);
        show($("mod-ok"), "Uploaded " + (out && out.name ? out.name : f.name) + ".");
      } catch (e) {
        $("mod-progress").hidden = true;
        show($("mod-err"), e.message);
      } finally {
        if (btn) btn.disabled = false;
      }
    };

    const FORCE_COUNTDOWN_SEC = 10;
    const CRON_MONTHS = { jan:1,feb:2,mar:3,apr:4,may:5,jun:6,jul:7,aug:8,sep:9,oct:10,nov:11,dec:12 };
    const CRON_DOW = { sun:0,mon:1,tue:2,wed:3,thu:4,fri:5,sat:6 };
    const CRON_DESCRIPTORS = { "@yearly":1,"@annually":1,"@monthly":1,"@weekly":1,"@daily":1,"@midnight":1,"@hourly":1 };
    const KNOWN_NOTIFY_TOKENS = { version:1, image:1, remaining:1 };
    const IANA_ZONE = /^(UTC|GMT|Etc\/[A-Za-z0-9+_:-]+|[A-Z][A-Za-z]+(?:\/[A-Za-z0-9_+\-]+)+)$/;
    const OCI_REPO = /^[a-z0-9]+(?:[._-][a-z0-9]+)*(?::[0-9]+)?(?:\/[a-z0-9]+(?:[._-][a-z0-9]+)*)+$/i;
    const FIELD_ERR = {
      checkInterval: ["upd-interval","err-interval"],
      checkSchedule: ["upd-check-cron","err-check-cron"],
      applySchedule: ["upd-apply-cron","err-apply-cron"],
      timeZone: ["upd-tz","err-tz"],
      imageRepository: ["upd-repo","err-repo"],
      notifySchedule: ["upd-notify-sched","err-notify-sched"]
    };
    let updateState = { pinned: "—", latest: "—", latestImage: "", updateAvailable: false };
    let savedUpdate = {};
    let forceRunning = false;
    let forceTimer = null;

    function parseGoDurationMs(raw) {
      const s = String(raw || "").trim();
      if (!s) return { empty: true };
      const m = s.match(/^([+-])?((?:(?:\d+(?:\.\d+)?|\.\d+)(?:ns|us|µs|μs|ms|s|m|h))+)$/);
      if (!m) return { error: true };
      const sign = m[1] === "-" ? -1 : 1;
      const body = m[2];
      const partRe = /(?:\d+(?:\.\d+)?|\.\d+)(?:ns|us|µs|μs|ms|s|m|h)/g;
      let ms = 0;
      let part;
      while ((part = partRe.exec(body))) {
        const tok = part[0];
        const um = tok.match(/^((?:\d+(?:\.\d+)?|\.\d+))(ns|us|µs|μs|ms|s|m|h)$/);
        const n = Number(um[1]);
        const unit = um[2];
        const mul = unit === "h" ? 3600000 : unit === "m" ? 60000 : unit === "s" ? 1000 : unit === "ms" ? 1 : 0.001;
        ms += n * mul;
      }
      ms *= sign;
      if (!Number.isFinite(ms) || ms <= 0) return { error: true };
      return { ms };
    }
    function cronNameOrNum(tok, min, max, names) {
      const low = tok.toLowerCase();
      if (names && names[low] != null) return names[low];
      if (!/^\d+$/.test(tok)) return null;
      const n = Number(tok);
      if (n < min || n > max) return null;
      return n;
    }
    function validCronAtom(atom, min, max, names) {
      const stepParts = atom.split("/");
      if (stepParts.length > 2) return false;
      const range = stepParts[0];
      if (stepParts.length === 2 && !/^[1-9]\d*$/.test(stepParts[1])) return false;
      if (range === "*") return true;
      const ends = range.split("-");
      if (ends.length > 2) return false;
      const a = cronNameOrNum(ends[0], min, max, names);
      if (a == null) return false;
      if (ends.length === 1) return true;
      const b = cronNameOrNum(ends[1], min, max, names);
      return b != null && b >= a;
    }
    function validCronField(field, min, max, names) {
      return !!field && field.split(",").every(function(atom) { return validCronAtom(atom, min, max, names); });
    }
    function validCronExpr(raw) {
      const expr = String(raw || "").trim();
      if (!expr) return true;
      const low = expr.toLowerCase();
      if (CRON_DESCRIPTORS[low]) return true;
      if (low.startsWith("@every ")) {
        const d = parseGoDurationMs(expr.slice(6));
        return !d.empty && !d.error;
      }
      if (expr.startsWith("@")) return false;
      const parts = expr.split(/\s+/);
      return parts.length === 5
        && validCronField(parts[0], 0, 59, null)
        && validCronField(parts[1], 0, 23, null)
        && validCronField(parts[2], 1, 31, null)
        && validCronField(parts[3], 1, 12, CRON_MONTHS)
        && validCronField(parts[4], 0, 7, CRON_DOW);
    }
    function readUpdateForm() {
      return {
        autoUpdateImage: $("upd-auto").checked,
        checkInterval: $("upd-interval").value.trim(),
        checkSchedule: $("upd-check-cron").value.trim(),
        applySchedule: $("upd-apply-cron").value.trim(),
        timeZone: $("upd-tz").value.trim(),
        imageRepository: $("upd-repo").value.trim(),
        onlyWhenEmpty: $("upd-empty").checked,
        notifyPlayers: $("upd-notify").checked,
        notifySchedule: $("upd-notify-sched").value.trim(),
        notifyMessage: $("upd-notify-msg").value.trim()
      };
    }
    function writeUpdateForm(s) {
      $("upd-auto").checked = !!s.autoUpdateImage;
      $("upd-interval").value = s.checkInterval || "";
      $("upd-check-cron").value = s.checkSchedule || "";
      $("upd-apply-cron").value = s.applySchedule || "";
      $("upd-tz").value = s.timeZone || "";
      $("upd-repo").value = s.imageRepository || "";
      $("upd-empty").checked = s.onlyWhenEmpty !== false;
      $("upd-notify").checked = !!s.notifyPlayers;
      $("upd-notify-sched").value = s.notifySchedule || "";
      $("upd-notify-msg").value = s.notifyMessage || "";
      savedUpdate = readUpdateForm();
      clearUpdateFieldErrors();
      syncUpdateDirty();
    }
    function setFieldNote(inputId, noteId, msg, warn) {
      const input = $(inputId);
      const note = $(noteId);
      const field = input && input.closest(".field");
      if (note) { note.textContent = msg || ""; note.hidden = !msg; }
      if (field) field.classList.toggle("is-invalid", !!(msg && !warn));
      if (input) input.setAttribute("aria-invalid", msg && !warn ? "true" : "false");
    }
    function clearUpdateFieldErrors() {
      Object.keys(FIELD_ERR).forEach(function(k) {
        setFieldNote(FIELD_ERR[k][0], FIELD_ERR[k][1], "");
      });
      setFieldNote("upd-notify-msg", "warn-notify-msg", "", true);
    }
    function notifyMessageHint(s) {
      if (!s.includes("{")) return "";
      const tokens = [];
      const re = /\{([^{}]*)\}/g;
      let m;
      while ((m = re.exec(s))) tokens.push(m[1]);
      const unknown = tokens.filter(function(t) { return !KNOWN_NOTIFY_TOKENS[t]; });
      if (/\{[^{}]*$/.test(s.replace(/\{[^{}]*\}/g, ""))) return "Unclosed {";
      if (unknown.length) return "Unknown {" + unknown[0] + "}";
      return "";
    }
    function validateUpdateForm() {
      const s = readUpdateForm();
      let ok = true;
      const interval = parseGoDurationMs(s.checkInterval);
      setFieldNote("upd-interval", "err-interval", interval.error ? "Need a Go duration (6h, 1h30m)" : "");
      if (interval.error) ok = false;
      const cronCheck = s.checkSchedule && !validCronExpr(s.checkSchedule);
      setFieldNote("upd-check-cron", "err-check-cron", cronCheck ? "Need 5-field cron or @hourly" : "");
      if (cronCheck) ok = false;
      const cronApply = s.applySchedule && !validCronExpr(s.applySchedule);
      setFieldNote("upd-apply-cron", "err-apply-cron", cronApply ? "Need 5-field cron or @hourly" : "");
      if (cronApply) ok = false;
      const tzBad = s.timeZone && !IANA_ZONE.test(s.timeZone);
      setFieldNote("upd-tz", "err-tz", tzBad ? "Need IANA zone (UTC, America/Los_Angeles)" : "");
      if (tzBad) ok = false;
      const repoBad = s.imageRepository && (/\s/.test(s.imageRepository) || !OCI_REPO.test(s.imageRepository.replace(/\/+$/, "")));
      setFieldNote("upd-repo", "err-repo", repoBad ? "Need host/path" : "");
      if (repoBad) ok = false;
      let schedBad = false;
      if (s.notifySchedule) {
        schedBad = s.notifySchedule.split(",").some(function(t) {
          t = t.trim();
          return !t || parseGoDurationMs(t).error;
        });
      }
      setFieldNote("upd-notify-sched", "err-notify-sched", schedBad ? "Each item needs a Go duration" : "");
      if (schedBad) ok = false;
      setFieldNote("upd-notify-msg", "warn-notify-msg", notifyMessageHint(s.notifyMessage), true);
      return ok;
    }
    function syncUpdateDirty() {
      const dirty = JSON.stringify(readUpdateForm()) !== JSON.stringify(savedUpdate);
      const el = $("upd-dirty");
      if (el) el.hidden = !dirty;
    }
    function renderUpdateStatus() {
      const box = $("upd-stats");
      if (!box) return;
      box.replaceChildren(statCard("Pinned", updateState.pinned || "—"), statCard("Latest", updateState.latest || "—"));
      const badge = $("upd-badge");
      if (forceRunning) {
        badge.textContent = "Updating";
        badge.className = "badge badge--danger";
      } else if (updateState.updateAvailable) {
        badge.textContent = "Update ready";
        badge.className = "badge badge--warn";
      } else {
        badge.textContent = "Up to date";
        badge.className = "badge badge--ok";
      }
      const wrap = $("upd-force-wrap");
      if (wrap) wrap.hidden = !(updateState.updateAvailable || forceRunning);
      const btn = $("upd-force");
      if (btn) btn.disabled = forceRunning || !updateState.updateAvailable;
    }
    function applyUpdatesPayload(out) {
      if (!out) return;
      updateState = {
        pinned: out.pinned || "—",
        latest: out.latest || "—",
        latestImage: out.latestImage || "",
        updateAvailable: !!out.updateAvailable
      };
      if (out.settings) writeUpdateForm(out.settings);
      renderUpdateStatus();
    }
    async function loadUpdates() {
      show($("upd-err"), "");
      try {
        applyUpdatesPayload(await api("/api/updates"));
      } catch (e) { show($("upd-err"), e.message); }
    }
    function forceAnnounceText(sec) {
      const remaining = sec + "s";
      const ver = updateState.latest || "";
      const image = updateState.latestImage || ver;
      const tmpl = savedUpdate.notifyMessage;
      if (tmpl) {
        return tmpl.split("{version}").join(ver).split("{image}").join(image).split("{remaining}").join(remaining);
      }
      return "[Server] Update " + ver + " — restart in " + remaining;
    }
    function paintForceTick(sec) {
      $("upd-force-announce").textContent = forceAnnounceText(sec);
      $("upd-force-left").textContent = String(sec);
      $("upd-force-left-label").textContent = sec === 1 ? "second" : "seconds";
      $("upd-force-bar").style.width = (((FORCE_COUNTDOWN_SEC - sec) / FORCE_COUNTDOWN_SEC) * 100) + "%";
    }
    $("upd-form").addEventListener("input", function() { validateUpdateForm(); syncUpdateDirty(); });
    $("upd-form").addEventListener("change", function() { validateUpdateForm(); syncUpdateDirty(); });
    $("upd-form").onsubmit = async function(ev) {
      ev.preventDefault();
      if (forceRunning) return;
      show($("upd-err"), ""); show($("upd-ok"), "");
      if (!validateUpdateForm()) { show($("upd-err"), "Not saved."); return; }
      const s = readUpdateForm();
      try {
        const out = await api("/api/updates", {
          method: "PUT",
          headers: {"Content-Type":"application/json"},
          body: JSON.stringify({
            autoUpdateImage: s.autoUpdateImage,
            checkInterval: s.checkInterval,
            checkSchedule: s.checkSchedule,
            applySchedule: s.applySchedule,
            timeZone: s.timeZone,
            imageRepository: s.imageRepository,
            onlyWhenEmpty: s.onlyWhenEmpty,
            notifyPlayers: s.notifyPlayers,
            notifySchedule: s.notifySchedule,
            notifyMessage: s.notifyMessage
          })
        });
        applyUpdatesPayload(out);
        show($("upd-ok"), "Saved.");
      } catch (e) {
        if (e.fields) {
          Object.keys(e.fields).forEach(function(k) {
            if (FIELD_ERR[k]) setFieldNote(FIELD_ERR[k][0], FIELD_ERR[k][1], e.fields[k]);
          });
        }
        show($("upd-err"), e.message || "Not saved.");
      }
    };
    $("upd-reset").onclick = async function() {
      if (forceRunning) return;
      if (!confirm("Reset to defaults?")) return;
      show($("upd-err"), "");
      try {
        const out = await api("/api/updates/reset", { method: "POST" });
        applyUpdatesPayload(out);
        show($("upd-ok"), "Defaults restored.");
      } catch (e) { show($("upd-err"), e.message); }
    };
    $("upd-check").onclick = async function() {
      if (forceRunning) return;
      show($("upd-err"), ""); show($("upd-ok"), "");
      const btn = $("upd-check");
      btn.disabled = true;
      btn.textContent = "Checking…";
      try {
        applyUpdatesPayload(await api("/api/updates/check", { method: "POST" }));
      } catch (e) { show($("upd-err"), e.message); }
      btn.disabled = false;
      btn.textContent = "Check now";
    };
    $("upd-force").onclick = async function() {
      show($("upd-err"), ""); show($("upd-ok"), "");
      if (forceRunning || !updateState.updateAvailable) return;
      const image = updateState.latestImage || updateState.latest;
      if (!confirm("Save & force update to " + image + "?")) return;
      if (!confirm("Save, announce 10s, then Recreate?")) return;
      forceRunning = true;
      let left = FORCE_COUNTDOWN_SEC;
      $("upd-force-panel").hidden = false;
      paintChain("upd-force-chain", "save");
      $("upd-force-announce").textContent = "Saving…";
      $("upd-force-left").textContent = String(FORCE_COUNTDOWN_SEC);
      $("upd-force-left-label").textContent = "seconds";
      $("upd-force-bar").style.width = "0";
      renderUpdateStatus();
      forceTimer = setInterval(function() {
        left -= 1;
        if (left > 0) {
          paintChain("upd-force-chain", "announce");
          paintForceTick(left);
        }
      }, 1000);
      try {
        const out = await api("/api/updates/force", { method: "POST" });
        if (forceTimer) { clearInterval(forceTimer); forceTimer = null; }
        paintChain("upd-force-chain", "roll");
        document.querySelectorAll("#upd-force-chain .chain-step").forEach(function(el) {
          el.classList.remove("is-active");
          el.classList.add("is-done");
        });
        $("upd-force-left").textContent = "0";
        $("upd-force-bar").style.width = "100%";
        show($("upd-ok"), out.message || "Saved, announced, Recreate.");
        await loadUpdates();
      } catch (e) {
        show($("upd-err"), e.message);
      } finally {
        if (forceTimer) { clearInterval(forceTimer); forceTimer = null; }
        forceRunning = false;
        $("upd-force-panel").hidden = true;
        renderUpdateStatus();
      }
    };

    const PROFILE_DEFAULTS = {
      name: "", description: "", maxPlayers: 4,
      steam: true, xbox: true, ps5: true, mac: true, community: false
    };
    const OPT_GROUPS = [
      { title: "Performances", keys: [
        ["BaseCampMaxNum", "int"], ["BaseCampMaxNumInGuild", "int"],
        ["BaseCampWorkerMaxNum", "int"], ["MaxBuildingLimitNum", "int"]
      ]},
      { title: "Server management", keys: [
        ["bAllowClientMod", "bool"], ["bIsShowJoinLeftMessage", "bool"],
        ["bIsUseBackupSaveData", "bool"], ["bEnableBuildingPlayerUIdDisplay", "bool"]
      ]},
      { title: "Features", keys: [
        ["bExistPlayerAfterLogout", "bool"], ["bEnableInvaderEnemy", "bool"],
        ["bIsPvP", "bool"], ["bEnableFastTravel", "bool"],
        ["bEnableFastTravelOnlyBaseCamp", "bool"], ["bHardcore", "bool"],
        ["bCharacterRecreateInHardcore", "bool"], ["bAllowGlobalPalboxExport", "bool"],
        ["bAllowGlobalPalboxImport", "bool"], ["bShowPlayerList", "bool"],
        ["bEnableVoiceChat", "bool"], ["RandomizerType", "randomizer"],
        ["bIsStartLocationSelectByMap", "bool"]
      ]},
      { title: "Game balances", keys: [
        ["DayTimeSpeedRate", "num"], ["NightTimeSpeedRate", "num"], ["ExpRate", "num"],
        ["PalCaptureRate", "num"], ["PalSpawnNumRate", "num"], ["WorkSpeedRate", "num"],
        ["PalEggDefaultHatchingTime", "num"], ["CollectionDropRate", "num"],
        ["EnemyDropItemRate", "num"], ["PlayerDamageRateAttack", "num"],
        ["PlayerDamageRateDefense", "num"], ["PlayerStomachDecreaceRate", "num"],
        ["PlayerStaminaDecreaceRate", "num"], ["FishingDifficultyRate", "num"],
        ["ItemWeightRate", "num"], ["GuildPlayerMaxNum", "int"],
        ["GuildRejoinCooldownMinutes", "int"], ["SupplyDropSpan", "int"],
        ["DeathPenalty", "death"], ["bPalLost", "bool"], ["BlockRespawnTime", "int"]
      ]}
    ];
    let savedProfile = { ...PROFILE_DEFAULTS };
    let savedOptions = {};
    let liveSettings = {};
    let credOnce = { join: "", admin: "" };
    let credCopied = { join: false, admin: false };

    function readProfile() {
      return {
        name: $("prof-name").value.trim(),
        description: $("prof-desc").value.trim(),
        maxPlayers: Number($("prof-max").value),
        steam: $("prof-steam").checked,
        xbox: $("prof-xbox").checked,
        ps5: $("prof-ps5").checked,
        mac: $("prof-mac").checked,
        community: $("prof-community").checked
      };
    }
    function writeProfile(p) {
      $("prof-name").value = p.name || "";
      $("prof-desc").value = p.description || "";
      $("prof-max").value = p.maxPlayers;
      $("prof-steam").checked = !!p.steam;
      $("prof-xbox").checked = !!p.xbox;
      $("prof-ps5").checked = !!p.ps5;
      $("prof-mac").checked = !!p.mac;
      $("prof-community").checked = !!p.community;
      setFieldNote("prof-max", "err-prof-max", "");
      syncProfileDirty();
    }
    function validateProfile() {
      const n = Number($("prof-max").value);
      if (!Number.isInteger(n) || n < 1 || n > 32) {
        setFieldNote("prof-max", "err-prof-max", "1–32");
        return false;
      }
      setFieldNote("prof-max", "err-prof-max", "");
      return true;
    }
    function syncProfileDirty() {
      const el = $("prof-dirty");
      if (el) el.hidden = JSON.stringify(readProfile()) !== JSON.stringify(savedProfile);
    }
    function optControl(key, kind) {
      if (kind === "bool" || kind === "death" || kind === "randomizer") {
        const s = document.createElement("select");
        s.id = "opt-" + key;
        const opts = kind === "bool" ? [["", "inherit"], ["True", "True"], ["False", "False"]]
          : kind === "death" ? [["", "inherit"], ["None", "None"], ["Item", "Item"], ["ItemAndEquipment", "ItemAndEquipment"], ["All", "All"]]
          : [["", "inherit"], ["None", "None"], ["Region", "Region"], ["All", "All"]];
        opts.forEach(function(pair) {
          const o = document.createElement("option");
          o.value = pair[0];
          o.textContent = pair[1];
          s.appendChild(o);
        });
        return s;
      }
      const i = document.createElement("input");
      i.id = "opt-" + key;
      i.type = "text";
      i.inputMode = "decimal";
      i.placeholder = "inherit";
      i.autocomplete = "off";
      return i;
    }
    function buildOptForm() {
      const form = $("opt-form");
      if (!form || form.dataset.ready) return;
      form.dataset.ready = "1";
      OPT_GROUPS.forEach(function(g) {
        const h = document.createElement("h3");
        h.className = "opt-group";
        h.textContent = g.title;
        form.appendChild(h);
        g.keys.forEach(function(pair) {
          const key = pair[0], kind = pair[1];
          const row = document.createElement("div");
          row.className = "opt-row";
          row.dataset.key = key;
          const lab = document.createElement("label");
          lab.setAttribute("for", "opt-" + key);
          lab.textContent = key;
          const live = document.createElement("span");
          live.className = "opt-live";
          live.textContent = liveSettings[key] == null ? "—" : liveSettings[key];
          row.append(lab, live, optControl(key, kind));
          form.appendChild(row);
        });
      });
    }
    function paintLiveSettings() {
      document.querySelectorAll("#opt-form .opt-row").forEach(function(row) {
        const key = row.dataset.key;
        const live = row.querySelector(".opt-live");
        if (live) live.textContent = liveSettings[key] == null ? "—" : liveSettings[key];
      });
    }
    function readOptions() {
      const out = {};
      OPT_GROUPS.forEach(function(g) {
        g.keys.forEach(function(pair) {
          const el = $("opt-" + pair[0]);
          if (!el) return;
          const v = String(el.value || "").trim();
          if (v) out[pair[0]] = v;
        });
      });
      return out;
    }
    function writeOptions(map) {
      OPT_GROUPS.forEach(function(g) {
        g.keys.forEach(function(pair) {
          const el = $("opt-" + pair[0]);
          if (el) el.value = (map && map[pair[0]]) || "";
        });
      });
      document.querySelectorAll("#opt-form .opt-row.is-invalid").forEach(function(r) { r.classList.remove("is-invalid"); });
      syncOptDirty();
    }
    function validOptNumber(raw, intOnly) {
      if (!raw) return true;
      if (!/^-?\d+(\.\d+)?$/.test(raw)) return false;
      const n = Number(raw);
      if (!Number.isFinite(n) || n < 0) return false;
      if (intOnly && !Number.isInteger(n)) return false;
      return true;
    }
    function validateOptions() {
      let ok = true;
      OPT_GROUPS.forEach(function(g) {
        g.keys.forEach(function(pair) {
          const key = pair[0], kind = pair[1];
          const el = $("opt-" + key);
          const row = el && el.closest(".opt-row");
          const v = el ? String(el.value || "").trim() : "";
          const bad = (kind === "num" && !validOptNumber(v, false)) || (kind === "int" && !validOptNumber(v, true));
          if (row) row.classList.toggle("is-invalid", bad);
          if (el) el.setAttribute("aria-invalid", bad ? "true" : "false");
          if (bad) ok = false;
        });
      });
      return ok;
    }
    function syncOptDirty() {
      const el = $("opt-dirty");
      if (el) el.hidden = JSON.stringify(readOptions()) !== JSON.stringify(savedOptions);
    }
    function paintCred(kind, set) {
      const badge = $("cred-" + kind + "-badge");
      if (!badge) return;
      badge.textContent = set ? "Set" : "Missing";
      badge.className = "badge " + (set ? "badge--ok" : "badge--off");
    }
    async function loadSettings() {
      show($("set-err"), "");
      buildOptForm();
      try {
        const data = await api("/api/settings");
        savedProfile = Object.assign({}, PROFILE_DEFAULTS, data.profile || {});
        savedOptions = data.options || {};
        liveSettings = data.live || {};
        writeProfile(savedProfile);
        writeOptions(savedOptions);
        paintLiveSettings();
        const creds = data.credentials || {};
        paintCred("join", !!(creds.join && creds.join.set));
        paintCred("admin", !!(creds.admin && creds.admin.set));
        validateProfile();
        validateOptions();
      } catch (e) { show($("set-err"), e.message); }
    }
    $("prof-form").addEventListener("input", function() { validateProfile(); syncProfileDirty(); });
    $("prof-form").addEventListener("change", function() { validateProfile(); syncProfileDirty(); });
    $("prof-reset").onclick = function() {
      if (!confirm("Reset profile to defaults?")) return;
      writeProfile({ ...PROFILE_DEFAULTS });
      show($("set-err"), "");
      show($("set-ok"), "");
    };
    $("opt-form").addEventListener("input", function() { validateOptions(); syncOptDirty(); });
    $("opt-form").addEventListener("change", function() { validateOptions(); syncOptDirty(); });
    $("opt-reset").onclick = function() {
      if (!confirm("Reset game settings to inherit?")) return;
      writeOptions({});
      show($("set-err"), "");
      show($("set-ok"), "");
    };
    $("set-apply").onclick = async function() {
      show($("set-err"), ""); show($("set-ok"), "");
      if (!validateProfile() || !validateOptions()) {
        show($("set-err"), "Not applied.");
        return;
      }
      if (!confirm("Apply server profile and game settings, then Recreate?")) return;
      try {
        const out = await api("/api/settings", {
          method: "PUT",
          headers: {"Content-Type":"application/json"},
          body: JSON.stringify({ profile: readProfile(), options: readOptions() })
        });
        savedProfile = readProfile();
        savedOptions = readOptions();
        syncProfileDirty();
        syncOptDirty();
        show($("set-ok"), out.message || "Profile and game settings applied. Recreate requested.");
      } catch (e) { show($("set-err"), e.message); }
    };
    function copyOnce(kind) {
      const val = credOnce[kind];
      if (!val || credCopied[kind]) return;
      credCopied[kind] = true;
      const once = $("cred-" + kind + "-once");
      const mask = $("cred-" + kind + "-mask");
      const btn = $("cred-" + kind + "-copy");
      if (once) { once.hidden = false; once.textContent = val; }
      if (mask) mask.hidden = true;
      if (btn) { btn.textContent = "Copied"; btn.disabled = true; }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(val).catch(function() {});
      }
      show($("set-err"), "");
      show($("set-ok"), "Copied once.");
      setTimeout(function() {
        if (once) { once.textContent = ""; once.hidden = true; }
        if (mask) mask.hidden = false;
      }, 4000);
    }
    $("cred-join-copy").onclick = function() { copyOnce("join"); };
    $("cred-admin-copy").onclick = function() { copyOnce("admin"); };
    async function rotateCred(kind) {
      const label = kind === "join" ? "join" : "admin";
      if (!confirm("Rotate " + label + " password, then Recreate?")) return;
      if (!confirm("Rotate now? Old password stops working after Recreate.")) return;
      show($("set-err"), ""); show($("set-ok"), "");
      try {
        const out = await api("/api/credentials/rotate", {
          method: "POST",
          headers: {"Content-Type":"application/json"},
          body: JSON.stringify({ key: kind })
        });
        credOnce[kind] = out.password || "";
        credCopied[kind] = false;
        const once = $("cred-" + kind + "-once");
        const mask = $("cred-" + kind + "-mask");
        const copy = $("cred-" + kind + "-copy");
        if (once) { once.textContent = ""; once.hidden = true; }
        if (mask) { mask.hidden = false; mask.textContent = "••••••••"; }
        if (copy) { copy.textContent = "Copy"; copy.disabled = !credOnce[kind]; }
        paintCred(kind, true);
        show($("set-ok"), out.message || ("Rotated " + label + ". Recreate requested. Copy once if needed."));
      } catch (e) { show($("set-err"), e.message); }
    }
    $("cred-join-rotate").onclick = function() { rotateCred("join"); };
    $("cred-admin-rotate").onclick = function() { rotateCred("admin"); };
    $("cred-join-copy").disabled = true;
    $("cred-admin-copy").disabled = true;

    buildOptForm();
    writeProfile(savedProfile);
    writeOptions(savedOptions);
    syncRestartLabel();

    refreshStats();
    statsTimer = setInterval(() => {
      if (document.getElementById("overview").classList.contains("active") && !document.hidden) refreshStats();
    }, 8000);
  </script>
</body>
</html>
`
