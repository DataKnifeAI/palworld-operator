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
  <title>Palworld Mod Manager</title>
  <style>
    :root { --ink:#1c2e28; --muted:#3d554c; --paper:#f2f7f4; --sand:#fff3d6; --teal:#2aa8a0; --coral:#c44b28; --line:rgba(28,46,40,.12); }
    * { box-sizing: border-box; }
    body { margin:0; font-family: system-ui, sans-serif; background: var(--paper); color: var(--ink); }
    header { background: #1a6f78; color: #fff; padding: 1rem 1.25rem; }
    header h1 { margin:0; font-size:1.25rem; }
    header p { margin:.35rem 0 0; opacity:.9; font-size:.9rem; }
    main { max-width: 56rem; margin: 0 auto; padding: 1.25rem; }
    .warn { background: var(--sand); border: 1px solid var(--line); border-radius:.5rem; padding:.75rem 1rem; margin-bottom:1rem; font-size:.9rem; }
    nav { display:flex; flex-wrap:wrap; gap:.5rem; margin-bottom:1rem; }
    button, .btn { cursor:pointer; border:0; border-radius:.4rem; padding:.45rem .8rem; font: inherit; }
    .btn { background: var(--teal); color:#fff; text-decoration:none; display:inline-block; }
    .btn-danger { background: var(--coral); color:#fff; }
    .btn-ghost { background:#fff; border:1px solid var(--line); color: var(--ink); }
    table { width:100%; border-collapse: collapse; background:#fff; border-radius:.5rem; overflow:hidden; }
    th, td { text-align:left; padding:.55rem .75rem; border-bottom:1px solid var(--line); }
    th { background: #e8f4f2; font-size:.8rem; text-transform:uppercase; letter-spacing:.03em; }
    .muted { color: var(--muted); font-size:.85rem; }
    .row { display:flex; flex-wrap:wrap; gap:.75rem; align-items:end; margin:1rem 0; }
    label { display:block; font-size:.8rem; color: var(--muted); margin-bottom:.25rem; }
    input[type=file] { font: inherit; }
    .err { color: var(--coral); margin:.5rem 0; }
  </style>
</head>
<body>
  <header>
    <h1>Palworld Mod Manager</h1>
    <p>Authenticated admin UI for the mods PVC. Game UDP is unchanged.</p>
  </header>
  <main>
    <div class="warn">
      Official Linux PalServer does <strong>not</strong> load Pocketpair Workshop /
      <code>PalModSettings.ini</code>. Community <code>.pak</code> files go under
      <code>paks/~WorkshopMods</code> and <code>paks/LogicMods</code>.
      Restart uses Deployment Recreate — players disconnect until Ready.
      <a href="https://docs.palworldgame.com/settings-and-operation/mod/">Pocketpair mods (Windows)</a>
      ·
      <a href="https://yorkhost.fr/docs/en/palworld/mods-ue4ss">Yorkhost PAK / UE4SS</a>
    </div>
    <nav>
      <button class="btn-ghost" type="button" data-path="">PVC root (Mods)</button>
      <button class="btn-ghost" type="button" data-path="paks/~WorkshopMods">paks/~WorkshopMods</button>
      <button class="btn-ghost" type="button" data-path="paks/LogicMods">paks/LogicMods</button>
      <button class="btn-ghost" type="button" data-path="Workshop">Workshop</button>
      <button class="btn-danger" type="button" id="restart">Restart server</button>
    </nav>
    <p class="muted" id="crumb"></p>
    <div class="err" id="error"></div>
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
  </main>
  <script>
    const $ = (id) => document.getElementById(id);
    let current = "";
    const opts = { credentials: "same-origin" };
    function showErr(m) { $("error").textContent = m || ""; }
    async function api(url, init) {
      const r = await fetch(url, Object.assign({}, opts, init));
      if (r.status === 401) { showErr("Unauthorized"); throw new Error("401"); }
      const ct = r.headers.get("content-type") || "";
      const body = ct.includes("json") ? await r.json() : await r.text();
      if (!r.ok) {
        const msg = (body && body.error) ? body.error : r.statusText;
        showErr(msg);
        throw new Error(msg);
      }
      return body;
    }
    function joinPath(dir, name) {
      if (!dir) return name;
      return dir.replace(/\/$/, "") + "/" + name;
    }
    async function list(path) {
      current = path || "";
      $("crumb").textContent = "Path: /" + (current || "");
      showErr("");
      const q = encodeURIComponent(current);
      const data = await api("/api/files?path=" + q);
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
      await api("/api/upload", { method: "POST", body: fd });
      $("file").value = "";
      list(current);
    };
    $("restart").onclick = async () => {
      if (!confirm("Restart the Palworld server? Players will disconnect. Recreate strategy — downtime until the pod is Ready again. The mod manager UI will also restart.")) return;
      const out = await api("/api/restart", { method: "POST" });
      showErr(out.message || "Restart requested.");
    };
    list("");
  </script>
</body>
</html>
`
