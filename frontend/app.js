(function() {
  // ── SVG icons ──────────────────────────────────────────────────
  var ICON_COPY = '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="5.5" y="5.5" width="8" height="8" rx="1.5"/><path d="M3.5 10.5h-1a1 1 0 0 1-1-1v-7a1 1 0 0 1 1-1h7a1 1 0 0 1 1 1v1"/></svg>';
  var ICON_CHECK = '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3.5 8.5 6.5 11.5 12.5 4.5"/></svg>';
  var ICON_TOC = '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2.5 4h11M2.5 8h11M2.5 12h7"/></svg>';
  var ICON_CLOSE = '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3l10 10M13 3L3 13"/></svg>';

  var ADMONITION_ICONS = {
    note:      '<svg viewBox="0 0 16 16" fill="currentColor"><path d="M0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8Zm8-6.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13ZM6.5 7.75A.75.75 0 0 1 7.25 7h1a.75.75 0 0 1 .75.75v2.75h.25a.75.75 0 0 1 0 1.5h-2a.75.75 0 0 1 0-1.5h.25v-2h-.25a.75.75 0 0 1-.75-.75ZM8 6a1 1 0 1 1 0-2 1 1 0 0 1 0 2Z"/></svg>',
    tip:       '<svg viewBox="0 0 16 16" fill="currentColor"><path d="M8 1.5c-2.363 0-4 1.69-4 3.75 0 .984.424 1.625.984 2.304l.214.253c.223.264.47.556.673.848.284.411.537.896.621 1.49a.75.75 0 0 1-1.484.211c-.04-.282-.163-.547-.37-.847a8.456 8.456 0 0 0-.542-.68c-.084-.1-.173-.205-.268-.32C3.201 7.75 2.5 6.766 2.5 5.25 2.5 2.31 4.863.5 8 .5s5.5 1.81 5.5 4.75c0 1.516-.701 2.5-1.328 3.259-.095.115-.184.22-.268.32-.207.245-.383.453-.541.681-.208.3-.33.565-.37.847a.751.751 0 0 1-1.485-.212c.084-.593.337-1.078.621-1.489.203-.292.45-.584.673-.848.075-.088.147-.173.213-.253.561-.679.985-1.32.985-2.304 0-2.06-1.637-3.75-4-3.75ZM5.75 12h4.5a.75.75 0 0 1 0 1.5h-4.5a.75.75 0 0 1 0-1.5ZM6 15.25a.75.75 0 0 1 .75-.75h2.5a.75.75 0 0 1 0 1.5h-2.5a.75.75 0 0 1-.75-.75Z"/></svg>',
    important: '<svg viewBox="0 0 16 16" fill="currentColor"><path d="M0 1.75C0 .784.784 0 1.75 0h12.5C15.216 0 16 .784 16 1.75v9.5A1.75 1.75 0 0 1 14.25 13H8.06l-2.573 2.573A1.458 1.458 0 0 1 3 14.543V13H1.75A1.75 1.75 0 0 1 0 11.25Zm1.75-.25a.25.25 0 0 0-.25.25v9.5c0 .138.112.25.25.25h2a.75.75 0 0 1 .75.75v2.19l2.72-2.72a.749.749 0 0 1 .53-.22h6.5a.25.25 0 0 0 .25-.25v-9.5a.25.25 0 0 0-.25-.25Zm7 2.25v2.5a.75.75 0 0 1-1.5 0v-2.5a.75.75 0 0 1 1.5 0ZM9 9a1 1 0 1 1-2 0 1 1 0 0 1 2 0Z"/></svg>',
    warning:   '<svg viewBox="0 0 16 16" fill="currentColor"><path d="M6.457 1.047c.659-1.234 2.427-1.234 3.086 0l6.082 11.378A1.75 1.75 0 0 1 14.082 15H1.918a1.75 1.75 0 0 1-1.543-2.575Zm1.763.707a.25.25 0 0 0-.44 0L1.698 13.132a.25.25 0 0 0 .22.368h12.164a.25.25 0 0 0 .22-.368Zm.53 3.996v2.5a.75.75 0 0 1-1.5 0v-2.5a.75.75 0 0 1 1.5 0ZM9 11a1 1 0 1 1-2 0 1 1 0 0 1 2 0Z"/></svg>',
    caution:   '<svg viewBox="0 0 16 16" fill="currentColor"><path d="M4.47.22A.749.749 0 0 1 5 0h6c.199 0 .389.079.53.22l4.25 4.25c.141.14.22.331.22.53v6a.749.749 0 0 1-.22.53l-4.25 4.25A.749.749 0 0 1 11 16H5a.749.749 0 0 1-.53-.22L.22 11.53A.749.749 0 0 1 0 11V5c0-.199.079-.389.22-.53Zm.84 1.28L1.5 5.31v5.38l3.81 3.81h5.38l3.81-3.81V5.31L10.69 1.5ZM8 4a.75.75 0 0 1 .75.75v3.5a.75.75 0 0 1-1.5 0v-3.5A.75.75 0 0 1 8 4Zm0 8a1 1 0 1 1 0-2 1 1 0 0 1 0 2Z"/></svg>'
  };

  // ── WebSocket connection ──────────────────────────────────────
  var bufnr = new URLSearchParams(location.search).get("bufnr") || "1";
  var protocol = location.protocol === "https:" ? "wss:" : "ws:";
  var wsUrl = protocol + "//" + location.host + "/ws/?bufnr=" + bufnr;
  var ws;
  var contentEl = document.getElementById("content");
  var overlayEl = document.getElementById("folio-overlay");
  var statusTextEl = document.getElementById("folio-status-text");
  var statusDotEl = document.getElementById("folio-dot");
  var lightboxEl = document.getElementById("folio-lightbox");
  var lightboxImg = document.getElementById("folio-lightbox-img");
  var currentScrollLine = null;
  var reconnectDelay = 1000;

  function connect() {
    ws = new WebSocket(wsUrl);
    ws.onopen = function() {
      overlayEl.classList.remove("disconnected");
      statusTextEl.textContent = "folio · connected (buf " + bufnr + ")";
      statusDotEl.classList.add("connected");
      reconnectDelay = 1000;
    };
    ws.onmessage = function(ev) {
      try {
        var msg = JSON.parse(ev.data);
        handleMessage(msg);
      } catch (e) {
        console.error("folio: failed to parse message", e);
      }
    };
    ws.onclose = function() {
      overlayEl.classList.add("disconnected");
      statusTextEl.textContent = "folio · disconnected";
      statusDotEl.classList.remove("connected");
      reconnectDelay = Math.min(reconnectDelay * 2, 30000);
      setTimeout(connect, reconnectDelay);
    };
    ws.onerror = function() {
      ws.close();
    };
  }

  // ── Browser tab title ────────────────────────────────────────
  // Reflects the markdown file currently being previewed so multiple
  // preview tabs (one per Neovim buffer) are distinguishable, e.g.
  // "README.md — folio". Falls back to the static title when Neovim
  // hasn't reported a filename yet (e.g. an unsaved [No Name] buffer).
  var DEFAULT_TITLE = document.title;
  var currentTitleFilename = null;
  function updateTitle(filename) {
    filename = filename || "";
    if (filename === currentTitleFilename) return;
    currentTitleFilename = filename;
    document.title = filename ? filename + " — folio" : DEFAULT_TITLE;
  }

  function handleMessage(msg) {
    switch (msg.type) {
      case "render":
        updateTitle(msg.filename);
        if (msg.html) {
          currentScrollLine = null;  // reset — DOM is about to be replaced
          // Sanitize untrusted markdown HTML before injection. goldmark
          // runs with WithUnsafe() (to allow e.g. <details>), so raw
          // <script> / on* handlers could otherwise execute in the
          // browser. DOMPurify strips those while preserving
          // data-source-line and normal HTML.
          var safeHtml = (typeof DOMPurify !== 'undefined')
            ? DOMPurify.sanitize(msg.html, { ADD_ATTR: ['id'] })
            : msg.html;
          contentEl.innerHTML = safeHtml;
          window.enhanceContent(contentEl);
        }
        if (msg.scroll_to_line) {
          scrollToLine(msg.scroll_to_line);
        }
        break;
      case "scroll":
        if (msg.scroll_to_line) {
          scrollToLine(msg.scroll_to_line);
        }
        break;
    }
  }

  // ── Post-render enhancements ─────────────────────────────────

  // Exposed on window so static previews (render-test) can call it on load.
  window.enhanceContent = function enhanceContent(root) {
    // Each enhancement runs independently so a failure in one (e.g. a
    // malformed Mermaid block) cannot prevent the rest from rendering.
    var steps = [
      rewriteAssetUrls, addCopyButtons, highlightCode, processAdmonitions,
      wrapTables, markTaskListItems, renderMermaidDiagrams, renderMath,
      attachImageLightbox, buildTOC
    ];
    for (var i = 0; i < steps.length; i++) {
      try { steps[i](root); }
      catch (e) { console.warn('folio: enhancement step failed:', e); }
    }
  }

  // ── Asset URL rewriting ─────────────────────────────────────
  function rewriteAssetUrls(root) {
    root.querySelectorAll("img[src]").forEach(function(img) {
      var src = img.getAttribute("src");
      if (src && !/^(https?:|data:|#|\/)/.test(src)) {
        img.src = "/files/" + bufnr + "/" + src;
      }
    });
  }

  // ── Code copy buttons ───────────────────────────────────────
  function addCopyButtons(root) {
    root.querySelectorAll('pre > code').forEach(function(code) {
      var pre = code.parentElement;
      if (pre.querySelector('.folio-copy-btn')) return;
      var btn = document.createElement('button');
      btn.className = 'folio-copy-btn';
      btn.title = 'Copy code';
      btn.innerHTML = ICON_COPY;
      btn.addEventListener('click', function() {
        var text = code.textContent;
        navigator.clipboard.writeText(text).then(function() {
          btn.innerHTML = ICON_CHECK;
          btn.classList.add('copied');
          setTimeout(function() {
            btn.innerHTML = ICON_COPY;
            btn.classList.remove('copied');
          }, 2000);
        });
      });
      pre.appendChild(btn);
    });
  }

  // ── Syntax highlighting ─────────────────────────────────────
  function highlightCode(root) {
    if (typeof hljs === 'undefined') return;
    root.querySelectorAll('pre code[class*="language-"]').forEach(function(el) {
      hljs.highlightElement(el);
    });
  }

  // ── GitHub-flavored admonitions ─────────────────────────────
  function processAdmonitions(root) {
    root.querySelectorAll('blockquote').forEach(function(bq) {
      var firstP = bq.querySelector('p');
      if (!firstP) return;
      var text = firstP.innerHTML;
      var match = text.match(/^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*/i);
      if (!match) return;
      var type = match[1].toLowerCase();
      bq.classList.add('admonition-' + type);
      var icon = ADMONITION_ICONS[type] || '';
      var titleDiv = document.createElement('div');
      titleDiv.className = 'admonition-title';
      titleDiv.innerHTML = icon + ' ' + type.charAt(0).toUpperCase() + type.slice(1);
      // Remove the [!TYPE] prefix from the paragraph.
      firstP.innerHTML = text.replace(match[0], '');
      if (firstP.innerHTML.trim() === '') {
        firstP.remove();
      }
      bq.insertBefore(titleDiv, bq.firstChild);
    });
  }

  // ── Table wrapper for responsive scroll ─────────────────────
  function wrapTables(root) {
    root.querySelectorAll('table').forEach(function(table) {
      if (table.parentElement.classList.contains('table-wrapper')) return;
      var wrapper = document.createElement('div');
      wrapper.className = 'table-wrapper';
      table.parentElement.insertBefore(wrapper, table);
      wrapper.appendChild(table);
    });
  }

  // ── Task list item styling ──────────────────────────────────
  function markTaskListItems(root) {
    root.querySelectorAll('li > input[type="checkbox"]').forEach(function(cb) {
      cb.parentElement.classList.add('task-list-item');
      cb.setAttribute('disabled', '');
    });
  }

  // ── Mermaid diagram rendering ───────────────────────────────
  if (typeof mermaid !== 'undefined') {
    try {
      var mermaidTheme = (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches)
        ? 'dark' : 'default';
      mermaid.initialize({ startOnLoad: false, theme: mermaidTheme });
    } catch(e) {}
  }

  function renderMermaidDiagrams(root) {
    if (typeof mermaid === 'undefined') return;
    var candidates = root.querySelectorAll('pre code[class*="language-mermaid"]');
    if (candidates.length === 0) return;
    var toRender = [];
    candidates.forEach(function(code) {
      var pre = code.parentElement;
      var div = document.createElement('div');
      div.className = 'mermaid';
      div.textContent = code.textContent;
      pre.parentElement.replaceChild(div, pre);
      toRender.push(div);
    });
    if (toRender.length > 0) {
      try { mermaid.run({ nodes: toRender }); } catch(e) { console.warn('Mermaid error:', e); }
    }
  }

  // ── KaTeX math rendering ────────────────────────────────────
  function renderMath(root) {
    if (typeof renderMathInElement === 'undefined') return;
    try {
      renderMathInElement(root, {
        delimiters: [
          {left: '$$', right: '$$', display: true},
          {left: '$', right: '$', display: false},
          {left: '\\[', right: '\\]', display: true},
          {left: '\\(', right: '\\)', display: false}
        ],
        throwOnError: false
      });
    } catch(e) { console.warn('KaTeX render error:', e); }
  }

  // ── Image lightbox ──────────────────────────────────────────
  function attachImageLightbox(root) {
    root.querySelectorAll('img').forEach(function(img) {
      img.addEventListener('click', function(e) {
        e.preventDefault();
        lightboxImg.src = img.src;
        lightboxEl.classList.add('active');
      });
    });
  }
  lightboxEl.addEventListener('click', function() {
    lightboxEl.classList.remove('active');
    lightboxImg.src = '';
  });
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && lightboxEl.classList.contains('active')) {
      lightboxEl.classList.remove('active');
      lightboxImg.src = '';
    }
  });

  // ── Table of Contents (TOC) ──────────────────────────────────
  // Built from the rendered headings (h1–h6), each of which carries a
  // stable `id` attribute assigned by the Go renderer (GitHub-style slug,
  // de-duplicated per document). Clicking an entry smooth-scrolls to the
  // heading and updates the URL hash; scrolling the document highlights
  // the entry closest to (at or above) the viewport top.
  var tocToggleEl = document.getElementById('folio-toc-toggle');
  var tocPanelEl = document.getElementById('folio-toc-panel');
  var tocListEl = document.getElementById('folio-toc-list');
  var tocBackdropEl = document.getElementById('folio-toc-backdrop');
  var tocCloseEl = document.getElementById('folio-toc-close');
  var tocLinks = [];       // cached <a> elements, in document order
  var tocHeadings = [];    // matching heading elements, in document order
  var tocScrollRaf = null;

  function buildTOC(root) {
    var headings = Array.prototype.slice.call(
      root.querySelectorAll('h1[id], h2[id], h3[id], h4[id], h5[id], h6[id]')
    );

    if (headings.length === 0) {
      tocToggleEl.classList.remove('visible');
      closeTOC();
      tocListEl.innerHTML = '';
      tocLinks = [];
      tocHeadings = [];
      return;
    }

    var minLevel = 6;
    headings.forEach(function(h) {
      var lvl = parseInt(h.tagName.substring(1), 10);
      if (lvl < minLevel) minLevel = lvl;
    });

    var frag = document.createDocumentFragment();
    tocLinks = [];
    tocHeadings = headings;

    headings.forEach(function(h) {
      var lvl = parseInt(h.tagName.substring(1), 10);
      var li = document.createElement('li');
      var a = document.createElement('a');
      a.href = '#' + h.id;
      a.textContent = h.textContent;
      a.title = h.textContent;
      a.setAttribute('data-toc-level', String(Math.max(1, lvl - minLevel + 1)));
      a.setAttribute('data-toc-target', h.id);
      a.addEventListener('click', function(e) {
        e.preventDefault();
        navigateToHeading(h);
      });
      li.appendChild(a);
      frag.appendChild(li);
      tocLinks.push(a);
    });

    tocListEl.innerHTML = '';
    tocListEl.appendChild(frag);
    tocToggleEl.classList.add('visible');
    updateActiveTOCEntry();
  }

  function navigateToHeading(h) {
    history.replaceState(null, '', '#' + h.id);
    var rect = h.getBoundingClientRect();
    var targetTop = rect.top + window.pageYOffset - (window.innerHeight * 0.05);
    window.scrollTo({ top: Math.max(0, targetTop), behavior: 'smooth' });
    closeTOC();
  }

  function updateActiveTOCEntry() {
    if (tocHeadings.length === 0) return;
    // Find the last heading whose top is at or above a small offset from
    // the viewport top; that's the section currently being read.
    var activeIdx = 0;
    var threshold = 80; // px from top — accounts for scroll-margin-top feel
    for (var i = 0; i < tocHeadings.length; i++) {
      var top = tocHeadings[i].getBoundingClientRect().top;
      if (top - threshold <= 0) {
        activeIdx = i;
      } else {
        break;
      }
    }
    for (var j = 0; j < tocLinks.length; j++) {
      tocLinks[j].classList.toggle('active', j === activeIdx);
    }
    var activeLink = tocLinks[activeIdx];
    if (activeLink && tocPanelEl.classList.contains('open')) {
      var panelRect = tocListEl.getBoundingClientRect();
      var linkRect = activeLink.getBoundingClientRect();
      if (linkRect.top < panelRect.top || linkRect.bottom > panelRect.bottom) {
        activeLink.scrollIntoView({ block: 'center' });
      }
    }
  }

  function openTOC() {
    tocPanelEl.classList.add('open');
    tocBackdropEl.classList.add('open');
    tocToggleEl.classList.add('active');
    updateActiveTOCEntry();
  }

  function closeTOC() {
    tocPanelEl.classList.remove('open');
    tocBackdropEl.classList.remove('open');
    tocToggleEl.classList.remove('active');
  }

  function toggleTOC() {
    if (tocPanelEl.classList.contains('open')) {
      closeTOC();
    } else {
      openTOC();
    }
  }

  tocToggleEl.addEventListener('click', toggleTOC);
  tocCloseEl.addEventListener('click', closeTOC);
  tocBackdropEl.addEventListener('click', closeTOC);
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && tocPanelEl.classList.contains('open')) {
      closeTOC();
    }
  });
  window.addEventListener('scroll', function() {
    if (tocScrollRaf) return;
    tocScrollRaf = requestAnimationFrame(function() {
      tocScrollRaf = null;
      updateActiveTOCEntry();
    });
  }, { passive: true });

  // ── Scroll sync: anchor interpolation ───────────────────────

  function scrollToLine(line) {
    if (line === currentScrollLine) return;
    currentScrollLine = line;

    var prev = contentEl.querySelector(".folio-cursor-line");
    if (prev) prev.classList.remove("folio-cursor-line");

    var target = contentEl.querySelector('[data-source-line="' + line + '"]');

    if (!target) {
      var allBlocks = contentEl.querySelectorAll("[data-source-line]");
      var best = null;
      for (var i = 0; i < allBlocks.length; i++) {
        var l = parseInt(allBlocks[i].getAttribute("data-source-line"), 10);
        if (l <= line && (!best || l > best.l)) {
          best = { el: allBlocks[i], l: l };
        }
      }
      // Fallback: if no block precedes the target line, use the first block.
      if (!best && allBlocks.length > 0) {
        best = { el: allBlocks[0], l: parseInt(allBlocks[0].getAttribute("data-source-line"), 10) };
      }
      target = best ? best.el : null;
    }

    if (!target) return;

    target.classList.add("folio-cursor-line");

    // Compute the absolute position of the target within the document and
    // scroll there directly.  Using element position + pageYOffset is more
    // reliable than scrollIntoView, especially for large scroll distances.
    var rect = target.getBoundingClientRect();
    var targetTop = rect.top + window.pageYOffset;
    var margin = window.innerHeight * 0.1;
    window.scrollTo(0, Math.max(0, targetTop - margin));
  }

  // ── Bootstrap ───────────────────────────────────────────────
  connect();
})();
