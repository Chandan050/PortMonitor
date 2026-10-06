import "./style.css";

const root = document.querySelector("#app");

let ports = [];
let lastRefresh = "-";

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

import {
  ListPorts,
  KillProcess,
  RelaunchAsAdministrator,
  CurrentTime,
} from "../wailsjs/go/main/App.js";

async function invoke(method, ...args) {
  const methods = {
    ListPorts,
    KillProcess,
    RelaunchAsAdministrator,
    CurrentTime,
  };
  const fn = methods[method];
  if (!fn) throw new Error(`Unknown backend method: ${method}`);
  return await fn(...args);
}

function render() {
  root.innerHTML = `
    <main class="shell">
      <header class="header">
        <div>
          <div class="eyebrow">WINDOWS UTILITY</div>
          <h1>Port Monitor</h1>
          <p>View listening TCP ports and safely terminate the owning process.</p>
        </div>
        <button id="refresh" class="primary">↻ Refresh</button>
      </header>

      <section class="toolbar">
        <div class="search-wrap">
          <span>⌕</span>
          <input id="search" type="search" placeholder="Filter by port or process name..." />
        </div>
      </section>

      <section class="card">
        <div class="card-title">
          <span>Registered Ports</span>
          <span class="count">${filteredPorts().length} shown / ${ports.length} total</span>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Port</th>
                <th>Protocol</th>
                <th>PID</th>
                <th>Process</th>
                <th>Status</th>
                <th class="action-col">Action</th>
              </tr>
            </thead>
            <tbody id="rows">${rowsHtml()}</tbody>
          </table>
        </div>
      </section>

      <footer>
        <span id="message">Ready</span>
        <span>Last refreshed: ${escapeHtml(lastRefresh)}</span>
      </footer>
    </main>

    <div id="modal-root"></div>
  `;

  document.querySelector("#refresh").addEventListener("click", refresh);
  document.querySelector("#search").addEventListener("input", () => {
    document.querySelector("#rows").innerHTML = rowsHtml();
    bindKillButtons();
  });
  bindKillButtons();
}

function filteredPorts() {
  const q = document.querySelector("#search")?.value?.trim().toLowerCase() || "";
  if (!q) return ports;
  return ports.filter(p =>
    String(p.port).includes(q) ||
    String(p.pid).includes(q) ||
    p.processName.toLowerCase().includes(q)
  );
}

function rowsHtml() {
  const list = filteredPorts();
  if (!list.length) {
    return `<tr><td colspan="6" class="empty">No listening TCP ports found.</td></tr>`;
  }

  return list.map(p => `
    <tr>
      <td><strong>${escapeHtml(p.port)}</strong></td>
      <td>${escapeHtml(p.protocol)}</td>
      <td>${escapeHtml(p.pid)}</td>
      <td class="process">${escapeHtml(p.processName)}</td>
      <td><span class="status">${escapeHtml(p.status)}</span></td>
      <td><button class="kill" data-pid="${p.pid}" data-port="${p.port}" data-name="${escapeHtml(p.processName)}">Kill</button></td>
    </tr>
  `).join("");
}

function bindKillButtons() {
  document.querySelectorAll(".kill").forEach(button => {
    button.addEventListener("click", () => {
      const pid = Number(button.dataset.pid);
      const port = Number(button.dataset.port);
      const name = button.dataset.name;
      showConfirm(pid, port, name);
    });
  });
}

function showConfirm(pid, port, name) {
  const modal = document.querySelector("#modal-root");
  modal.innerHTML = `
    <div class="overlay">
      <div class="modal">
        <h2>Terminate process?</h2>
        <p>Are you sure you want to terminate <strong>${escapeHtml(name)}</strong> (PID: ${pid}), which is using port <strong>${port}</strong>?</p>
        <div class="modal-actions">
          <button id="cancel" class="secondary">Cancel</button>
          <button id="confirm-kill" class="danger">Kill Process</button>
        </div>
      </div>
    </div>
  `;

  modal.querySelector("#cancel").onclick = () => modal.innerHTML = "";
  modal.querySelector("#confirm-kill").onclick = async () => {
    modal.innerHTML = "";
    await kill(pid, port, name);
  };
}

async function kill(pid, port, name) {
  setMessage(`Terminating ${name} (PID ${pid})...`, "normal");

  try {
    const result = await invoke("KillProcess", pid, name, port);

    if (result.success) {
      setMessage(result.message, "success");
      await refresh(false);
      return;
    }

    if (result.requiresAdmin) {
      showAdminPrompt(result.message);
      return;
    }

    setMessage(result.message, "error");
  } catch (error) {
    setMessage(error.message || "Unexpected error.", "error");
  }
}

function showAdminPrompt(message) {
  const modal = document.querySelector("#modal-root");
  modal.innerHTML = `
    <div class="overlay">
      <div class="modal">
        <h2>Administrator permission required</h2>
        <p>${escapeHtml(message)}</p>
        <p class="muted">Windows will display its normal UAC prompt. This application never asks for or stores your Windows password.</p>
        <div class="modal-actions">
          <button id="admin-cancel" class="secondary">Cancel</button>
          <button id="run-admin" class="primary">Run as Administrator</button>
        </div>
      </div>
    </div>
  `;

  modal.querySelector("#admin-cancel").onclick = () => modal.innerHTML = "";
  modal.querySelector("#run-admin").onclick = async () => {
    try {
      await invoke("RelaunchAsAdministrator");
    } catch (error) {
      modal.innerHTML = "";
      setMessage(error.message || "Could not request administrator privileges.", "error");
    }
  };
}

async function refresh(showReady = true) {
  const button = document.querySelector("#refresh");
  if (button) button.disabled = true;

  try {
    const data = await invoke("ListPorts");
    ports = Array.isArray(data) ? data : [];
    lastRefresh = await invoke("CurrentTime");
    render();
    if (showReady) setMessage(`${ports.length} listening port${ports.length === 1 ? "" : "s"} found.`, "success");
  } catch (error) {
    setMessage(error.message || "Unable to scan listening ports.", "error");
  } finally {
    const refreshedButton = document.querySelector("#refresh");
    if (refreshedButton) refreshedButton.disabled = false;
  }
}

function setMessage(text, type = "normal") {
  const el = document.querySelector("#message");
  if (!el) return;
  el.textContent = text;
  el.className = type;
}

render();
