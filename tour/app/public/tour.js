let currentLesson = null;
let files = {};
let editors = {};

function makeFileId() {
  if (window.crypto && crypto.randomUUID) return 'file-' + crypto.randomUUID();
  return 'file-' + Math.random().toString(36).slice(2) + Date.now().toString(36);
}

function getEditorMode(filename) {
  if (filename.endsWith('.vuego')) return 'ace/mode/html';
  if (filename.endsWith('.json')) return 'ace/mode/json';
  if (filename.endsWith('.yml') || filename.endsWith('.yaml')) return 'ace/mode/yaml';
  if (filename.endsWith('.php')) return 'ace/mode/php';
  if (filename.endsWith('.sql') || filename.endsWith('.sqlite3')) return 'ace/mode/sql';
  return 'ace/mode/text';
}

function createEditor(container, filename, content) {
  const editor = ace.edit(container);
  editor.setTheme('ace/theme/github');
  editor.setShowPrintMargin(false);
  editor.setOptions({
    fontSize: '14px',
    tabSize: 2,
    useSoftTabs: true,
    maxLines: Infinity,
    minLines: 3
  });
  editor.setValue(content, -1);
  editor.session.setMode(getEditorMode(filename));
  editor.session.on('change', function() {
    files[filename] = editor.getValue();
  });
  return editor;
}

async function loadLesson(chapterSlug, lessonIdx) {
  const res = await fetch('/lesson/' + encodeURIComponent(chapterSlug) + '/' + encodeURIComponent(lessonIdx), {
    headers: { 'Accept': 'application/json' }
  });
  const data = await res.json();
  if (data.error) {
    console.error('Failed to load lesson:', data.error);
    return;
  }
  currentLesson = data;
  files = { ...currentLesson.Files };
  
  document.getElementById('lessonContent').innerHTML = typeof marked !== 'undefined' ? marked.parse(currentLesson.Content) : currentLesson.Content;
  document.getElementById('lessonTitle').textContent = currentLesson.Title;
  document.getElementById('progressText').innerHTML = 'Lesson ' + (currentLesson.LessonIdx + 1) + ' of ' + currentLesson.TotalInChapter + ' in ' + currentLesson.ChapterTitle;
  
  const navButtons = document.querySelector('.nav-buttons');
  let prevBtn = document.getElementById('prevBtn');
  const nextBtn = document.getElementById('nextBtn');
  
  if (currentLesson.HasPrev) {
    if (!prevBtn) {
      prevBtn = document.createElement('a');
      prevBtn.id = 'prevBtn';
      prevBtn.href = '#';
      prevBtn.className = 'btn';
      prevBtn.innerHTML = '&larr; Previous';
      prevBtn.onclick = function() { navigate('prev'); return false; };
      navButtons.insertBefore(prevBtn, nextBtn);
    }
  } else if (prevBtn) {
    prevBtn.remove();
  }
  
  if (!currentLesson.HasNext) {
    nextBtn.innerHTML = 'Finish';
    nextBtn.href = '/done';
    nextBtn.onclick = null;
  } else {
    nextBtn.innerHTML = 'Next &rarr;';
    nextBtn.href = '#';
    nextBtn.onclick = function() { navigate('next'); return false; };
  }
  
  renderEditors();
  render();
}

function renderEditors() {
  const container = document.getElementById('editorsContainer');
  container.innerHTML = '';
  editors = {};
  
  const fileNames = Object.keys(files).sort((a, b) => {
    if (a.endsWith('.up.sql') && !b.endsWith('.up.sql')) return -1;
    if (!a.endsWith('.up.sql') && b.endsWith('.up.sql')) return 1;
    return a.localeCompare(b);
  });
  
  fileNames.forEach((name, i) => {
    const options = (currentLesson.FileOptions && currentLesson.FileOptions[name]) || [];

    const section = document.createElement('div');
    section.className = 'file-section';
    section.dataset.filename = name;
    
    const header = document.createElement('div');
    header.className = 'file-header';
    header.innerHTML = '<span class="file-toggle">&darr;</span> ' + name;
    header.onclick = () => toggleFile(name);
    
    const editorWrapper = document.createElement('div');
    editorWrapper.className = 'file-editor';
    editorWrapper.id = 'editor-' + i;
    
    section.appendChild(header);
    section.appendChild(editorWrapper);
    container.appendChild(section);
    
    editors[name] = createEditor(editorWrapper, name, files[name] || '');

    if (options.includes('hidden')) {
      editorWrapper.classList.add('collapsed');
      header.querySelector('.file-toggle').innerHTML = '&rarr;';
      editors[name].resize();
    }
  });
}

function toggleFile(name) {
  const section = document.querySelector('.file-section[data-filename="' + cssEscape(name) + '"]');
  if (!section) return;
  const editorWrapper = section.querySelector('.file-editor');
  const toggle = section.querySelector('.file-toggle');
  
  if (editorWrapper.classList.contains('collapsed')) {
    editorWrapper.classList.remove('collapsed');
    toggle.innerHTML = '&darr;';
    if (editors[name]) {
      editors[name].resize();
    }
  } else {
    editorWrapper.classList.add('collapsed');
    toggle.innerHTML = '&rarr;';
  }
}

function cssEscape(value) {
  if (window.CSS && CSS.escape) return CSS.escape(value);
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}

function toggleOutput() {
  const content = document.getElementById('outputContent');
  const toggle = document.getElementById('outputToggle');
  
  if (content.classList.contains('collapsed')) {
    content.classList.remove('collapsed');
    toggle.innerHTML = '&darr;';
  } else {
    content.classList.add('collapsed');
    toggle.innerHTML = '&rarr;';
  }
}

function syncFiles() {
  for (const [name, editor] of Object.entries(editors)) {
    files[name] = editor.getValue();
  }
}

async function render() {
  syncFiles();

  let template = '';
  let data = '{}';
  
  for (const [name, content] of Object.entries(files)) {
    if (name.endsWith('.vuego')) {
      template = content;
    } else if (name.endsWith('.json') || name.endsWith('.yml') || name.endsWith('.yaml')) {
      data = content;
    }
  }

  if (!template) {
    for (const [name, content] of Object.entries(files)) {
      if (name.endsWith('.php') || name.endsWith('.sql')) {
        template = '';
        break;
      }
    }
  }

  const res = await fetch('/render', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ template, data, files })
  });
  
  const result = await res.json();
  const iframe = document.getElementById('output');
  iframe.srcdoc = result.error ? '<pre style="color:red;padding:1rem">' + result.error + '</pre>' : result.html;
}

function navigate(dir) {
  if (!currentLesson) return;
  
  if (dir === 'prev' && currentLesson.HasPrev) {
    const slug = currentLesson.PrevSlug;
    const idx = currentLesson.PrevLessonIdx;
    window.history.pushState({}, '', '/lesson/' + slug + '/' + idx);
    loadLesson(slug, idx);
  } else if (dir === 'next' && currentLesson.HasNext) {
    const slug = currentLesson.NextSlug;
    const idx = currentLesson.NextLessonIdx;
    window.history.pushState({}, '', '/lesson/' + slug + '/' + idx);
    loadLesson(slug, idx);
  }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
  const pathMatch = window.location.pathname.match(/\/lesson\/([^\/]+)\/(\d+)/);
  if (pathMatch) {
    loadLesson(pathMatch[1], pathMatch[2]);
  } else {
    // Index page - render README and hide editors
    const readmeEl = document.getElementById('readmeContent');
    if (readmeEl && readmeEl.dataset.markdown) {
      readmeEl.innerHTML = typeof marked !== 'undefined' ? marked.parse(readmeEl.dataset.markdown) : readmeEl.dataset.markdown;
    }
    const editorPanel = document.querySelector('.editor-panel');
    if (editorPanel) {
      editorPanel.style.display = 'none';
    }
    const lessonPanel = document.querySelector('.lesson-panel');
    if (lessonPanel) {
      lessonPanel.style.width = '100%';
    }
  }
});
