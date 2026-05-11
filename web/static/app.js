const notesList = document.getElementById("notes-list");
const preview = document.getElementById("preview");
const generateForm = document.getElementById("generate-form");
const refreshBtn = document.getElementById("refresh-notes");
const dailyBtn = document.getElementById("daily-btn");
const syncForm = document.getElementById("sync-form");
const syncGithubForm = document.getElementById("sync-github-form");
const syncResults = document.getElementById("sync-results");
const syncResultsSection = document.getElementById("sync-results-section");
const syncResultsTitle = document.getElementById("sync-results-title");
const syncResultsCount = document.getElementById("sync-results-count");
const sendAllBtn = document.getElementById("send-all-btn");
const refreshUsageBtn = document.getElementById("refresh-usage");
const usageBody = document.getElementById("usage-body");
const manualSection = document.getElementById("manual-section");
const showManualBtn = document.getElementById("show-manual-btn");
const closeManualBtn = document.getElementById("close-manual-btn");
const copyPreviewBtn = document.getElementById("copy-preview");

// Tabs
const tabGitLab = document.getElementById("tab-gitlab");
const tabGitHub = document.getElementById("tab-github");

function authHeaders() {
  const params = new URLSearchParams(window.location.search);
  const token = params.get("token");
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

// UI State Toggles
showManualBtn.onclick = () => manualSection.classList.remove("hidden");
closeManualBtn.onclick = () => manualSection.classList.add("hidden");

tabGitLab.onclick = () => {
    tabGitLab.classList.add("tab-active", "text-blue-600");
    tabGitLab.classList.remove("text-gray-400");
    tabGitHub.classList.remove("tab-active", "text-blue-600");
    tabGitHub.classList.add("text-gray-400");
    syncForm.classList.remove("hidden");
    syncGithubForm.classList.add("hidden");
};

tabGitHub.onclick = () => {
    tabGitHub.classList.add("tab-active", "text-blue-600");
    tabGitHub.classList.remove("text-gray-400");
    tabGitLab.classList.remove("tab-active", "text-blue-600");
    tabGitLab.classList.add("text-gray-400");
    syncGithubForm.classList.remove("hidden");
    syncForm.classList.add("hidden");
};

async function loadNotes() {
  const response = await fetch("/api/notes" + window.location.search, {
    headers: authHeaders(),
  });
  if (!response.ok) {
    notesList.innerHTML = `<li class="p-4 text-red-500">Failed to load notes (${response.status})</li>`;
    return;
  }

  const notes = await response.json();
  notesList.innerHTML = "";
  if (!notes.length) {
    notesList.innerHTML = '<li class="p-8 text-center text-gray-400 text-sm">No notes found</li>';
    return;
  }

  for (const note of notes) {
    const li = document.createElement("li");
    li.className = "p-4 hover:bg-gray-50 transition-colors flex items-center justify-between group";
    
    const info = document.createElement("div");
    info.className = "flex flex-col";
    
    const title = document.createElement("span");
    title.className = "font-medium text-gray-800";
    title.textContent = note.title;
    
    const meta = document.createElement("span");
    meta.className = "text-xs text-gray-400 mt-1";
    meta.textContent = note.filename;
    
    info.appendChild(title);
    info.appendChild(meta);
    
    const actions = document.createElement("div");
    actions.className = "opacity-0 group-hover:opacity-100 transition-opacity";
    
    const viewLink = document.createElement("a");
    viewLink.href = `/api/notes/${encodeURIComponent(note.filename)}${window.location.search}`;
    viewLink.target = "_blank";
    viewLink.className = "text-xs bg-blue-50 text-blue-600 px-3 py-1 rounded-full font-semibold hover:bg-blue-100 transition-colors";
    viewLink.textContent = "View RAW";
    
    actions.appendChild(viewLink);
    li.appendChild(info);
    li.appendChild(actions);
    notesList.appendChild(li);
  }
}

generateForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  const commitsRaw = document.getElementById("commits").value.trim();
  const author = document.getElementById("author").value.trim();
  const commits = commitsRaw
    ? commitsRaw.split("\n").map((line) => ({ message: line.trim(), author }))
    : [];

  const payload = {
    provider: "manual",
    title: document.getElementById("title").value.trim(),
    author,
    repository: document.getElementById("repository").value.trim(),
    branch: document.getElementById("branch").value.trim(),
    description: document.getElementById("description").value.trim(),
    commits,
  };

  const response = await fetch("/api/generate" + window.location.search, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    updatePreview(`Generation failed (${response.status})`);
    return;
  }

  const data = await response.json();
  updatePreview(data.note || "No note returned.");
  manualSection.classList.add("hidden");
  await loadNotes();
});

refreshBtn.addEventListener("click", loadNotes);

dailyBtn.addEventListener("click", async () => {
  dailyBtn.disabled = true;
  updatePreview("Generating daily report...");

  const response = await fetch("/api/generate-daily" + window.location.search, {
    method: "POST",
    headers: authHeaders(),
  });

  if (!response.ok) {
    if (response.status === 404) {
      updatePreview("No events found for today to generate a report.");
    } else {
      updatePreview(`Daily generation failed (${response.status})`);
    }
    dailyBtn.disabled = false;
    return;
  }

  const data = await response.json();
  updatePreview(data.note || "No note returned.");
  dailyBtn.disabled = false;
  await loadNotes();
});

async function handleSync(event, provider) {
    event.preventDefault();
    const isGitLab = provider === 'gitlab';
    const project = document.getElementById(isGitLab ? "sync-project" : "sync-github-repo").value.trim();
    const startDate = document.getElementById(isGitLab ? "sync-start-date" : "sync-github-start-date").value;
    const endDate = document.getElementById(isGitLab ? "sync-end-date" : "sync-github-end-date").value;
    const btn = document.getElementById(isGitLab ? "sync-btn" : "sync-github-btn");

    btn.disabled = true;
    syncResultsSection.classList.remove("hidden");
    syncResultsTitle.textContent = `Syncing ${isGitLab ? 'GitLab' : 'GitHub'}...`;
    syncResults.innerHTML = '<div class="p-8 text-center text-gray-400">Fetching data from API...</div>';

    const params = new URLSearchParams(window.location.search);
    if (isGitLab) {
        params.set("project", project);
    } else {
        params.set("repo", project);
    }
    if (startDate) params.set("start_date", startDate);
    if (endDate) params.set("end_date", endDate);

    const apiPath = isGitLab ? "/api/sync-gitlab" : "/api/sync-github";
    const response = await fetch(`${apiPath}?${params.toString()}`, {
        headers: authHeaders(),
    });

    btn.disabled = false;
    if (!response.ok) {
        syncResults.innerHTML = `<div class="p-8 text-center text-red-500 font-medium">Sync failed (${response.status})</div>`;
        sendAllBtn.classList.add("hidden");
        return;
    }

    const data = await response.json();
    renderSyncResults(data, isGitLab);
}

syncForm.onsubmit = (e) => handleSync(e, 'gitlab');
syncGithubForm.onsubmit = (e) => handleSync(e, 'github');

function renderSyncResults(data, isGitLab) {
    syncResultsTitle.textContent = isGitLab ? `GitLab: ${data.project}` : `GitHub: ${data.repo}`;
    syncResultsCount.textContent = `${data.fetched} found`;
    
    if (!data.events || data.events.length === 0) {
        syncResults.innerHTML = `<div class="p-8 text-center text-gray-500">No merged ${isGitLab ? 'MRs' : 'PRs'} found for the selected range.</div>`;
        sendAllBtn.classList.add("hidden");
        return;
    }

    sendAllBtn.classList.remove("hidden");
    sendAllBtn.onclick = async () => {
        sendAllBtn.disabled = true;
        updatePreview("Generating consolidated release notes...");
        
        const params = new URLSearchParams(window.location.search);
        // For consolidated, we use the date of the sync. 
        // If it was a range, generate-daily might need to support ranges too or we just use the startDate.
        // The backend dailyGenerateHandler uses the "date" param.
        params.set("date", data.start_date); 
        // Note: our current generate-daily only takes one date and finds all events for that date.
        // If the user synced a range, they might expect notes for the whole range.
        
        const genResponse = await fetch(`/api/generate-daily?${params.toString()}`, {
            method: "POST",
            headers: authHeaders(),
        });
        
        sendAllBtn.disabled = false;
        if (!genResponse.ok) {
            updatePreview(`Batch generation failed (${genResponse.status})`);
            return;
        }
        const genData = await genResponse.json();
        updatePreview(genData.note || "No note returned.");
        await loadNotes();
    };

    syncResults.innerHTML = "";
    data.events.forEach(evt => {
        const item = document.createElement("div");
        item.className = "py-4 flex flex-col space-y-2";
        
        const header = document.createElement("div");
        header.className = "flex items-start justify-between";
        
        const titleInfo = document.createElement("div");
        titleInfo.innerHTML = `<span class="text-blue-600 font-mono font-bold mr-2">${isGitLab ? '!' : '#'}${evt.number}</span> <span class="font-bold text-gray-800">${evt.title}</span>`;
        
        const authorInfo = document.createElement("div");
        authorInfo.className = "text-xs text-gray-400 mt-1";
        authorInfo.textContent = `by ${evt.author} • ${new Date(evt.merged_at).toLocaleDateString()}`;
        
        titleInfo.appendChild(authorInfo);
        
        const actionBtn = document.createElement("button");
        actionBtn.className = "text-xs bg-gray-100 hover:bg-blue-600 hover:text-white text-gray-600 px-3 py-1 rounded-lg font-bold transition-all";
        actionBtn.textContent = "Send to AI";
        actionBtn.onclick = async () => {
            actionBtn.disabled = true;
            updatePreview(`Generating release notes for ${isGitLab ? 'MR' : 'PR'}...`);
            
            const genResponse = await fetch("/api/generate" + window.location.search, {
                method: "POST",
                headers: { "Content-Type": "application/json", ...authHeaders() },
                body: JSON.stringify(evt),
            });
            
            actionBtn.disabled = false;
            if (!genResponse.ok) {
                updatePreview(`Generation failed (${genResponse.status})`);
                return;
            }
            const genData = await genResponse.json();
            updatePreview(genData.note || "No note returned.");
            await loadNotes();
        };

        header.appendChild(titleInfo);
        header.appendChild(actionBtn);
        
        const commitsList = document.createElement("div");
        commitsList.className = "bg-gray-50 rounded-lg p-3 text-xs text-gray-600 space-y-1";
        evt.commits.forEach(c => {
            const cLine = document.createElement("div");
            cLine.className = "flex items-start";
            cLine.innerHTML = `<span class="text-gray-300 mr-2">•</span> <span class="flex-1">${c.message}</span>`;
            commitsList.appendChild(cLine);
        });

        item.appendChild(header);
        item.appendChild(commitsList);
        syncResults.appendChild(item);
    });
}

function updatePreview(text) {
    preview.textContent = text;
    if (text && !text.startsWith("Generating") && !text.startsWith("No preview")) {
        copyPreviewBtn.classList.remove("hidden");
    } else {
        copyPreviewBtn.classList.add("hidden");
    }
}

copyPreviewBtn.onclick = () => {
    navigator.clipboard.writeText(preview.textContent);
    const originalText = copyPreviewBtn.textContent;
    copyPreviewBtn.textContent = "Copied!";
    setTimeout(() => copyPreviewBtn.textContent = originalText, 2000);
};

async function loadUsage() {
  const response = await fetch("/api/usage" + window.location.search, {
    headers: authHeaders(),
  });
  if (!response.ok) return;

  const logs = await response.json();
  usageBody.innerHTML = "";
  logs.reverse().slice(0, 10).forEach((entry) => {
    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td class="py-2 text-gray-800 font-medium">${entry.model}</td>
      <td class="py-2 text-gray-500">${entry.input_tokens + entry.output_tokens}</td>
      <td class="py-2 text-right font-mono text-gray-600">$${entry.cost_usd.toFixed(4)}</td>
    `;
    usageBody.appendChild(tr);
  });
}

refreshUsageBtn.addEventListener("click", loadUsage);

// Initialize
const todayStr = new Date().toISOString().split('T')[0];
document.getElementById("sync-start-date").value = todayStr;
document.getElementById("sync-end-date").value = todayStr;
document.getElementById("sync-github-start-date").value = todayStr;
document.getElementById("sync-github-end-date").value = todayStr;

loadNotes();
loadUsage();
