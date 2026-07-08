(function() {
  // ── SVG icons ──────────────────────────────────────────────────
  // Single source of truth for every icon in the app (UI chrome +
  // admonition markers). All icons share one visual language — 16x16,
  // stroke-based, 1.5px rounded strokes — so the interface reads as one
  // coherent icon set instead of mixing stroke and filled styles.
  // index.html intentionally ships empty <button> shells for the TOC
  // toggle/close buttons; injectStaticIcons() fills them in at startup so
  // there is exactly one place (this object) where each icon is defined.
  var ICONS = {
    copy:  '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="5.5" y="5.5" width="8" height="8" rx="1.5"/><path d="M3.5 10.5h-1a1 1 0 0 1-1-1v-7a1 1 0 0 1 1-1h7a1 1 0 0 1 1 1v1"/></svg>',
    check: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3.5 8.5 6.5 11.5 12.5 4.5"/></svg>',
    toc:   '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2.5 4h11M2.5 8h11M2.5 12h7"/></svg>',
    close: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3l10 10M13 3L3 13"/></svg>',

    // Admonition markers — same stroke language as the UI icons above
    // (previously these were filled GitHub Octicons, which visually
    // clashed with the stroke-based copy/TOC/close icons elsewhere).
    note:      '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="8" cy="8" r="6.5"/><path d="M8 10.5v-3.25"/><path d="M8 5.5h.01"/></svg>',
    tip:       '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M6 14.5h4"/><path d="M6.5 12.5h3"/><path d="M8 1.5a4.5 4.5 0 0 0-2.4 8.31c.44.28.9.9.9 1.44v.25h3v-.25c0-.53.46-1.16.9-1.44A4.5 4.5 0 0 0 8 1.5Z"/></svg>',
    important: '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M1.5 2.5h13v9h-8l-3 3v-3h-2Z"/><path d="M8 5.25v3"/><path d="M8 10.5h.01"/></svg>',
    warning:   '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M8.02 1.85a.6.6 0 0 1 1.03 0l6.1 10.6a.6.6 0 0 1-.52.9H2.44a.6.6 0 0 1-.52-.9Z"/><path d="M8 6.25v3"/><path d="M8 11.5h.01"/></svg>',
    caution:   '<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M5 1.5h6l3.5 3.5v6L11 14.5H5L1.5 11v-6Z"/><path d="M8 5.25v3"/><path d="M8 10.5h.01"/></svg>'
  };

  // Fill in the empty icon buttons shipped by index.html — keeps the icon
  // markup defined exactly once (in ICONS above) instead of duplicated
  // between this file and index.html.
  function injectStaticIcons() {
    var toc = document.getElementById('folio-toc-toggle');
    var close = document.getElementById('folio-toc-close');
    if (toc) toc.innerHTML = ICONS.toc;
    if (close) close.innerHTML = ICONS.close;
  }
  injectStaticIcons();

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
      btn.innerHTML = ICONS.copy;
      btn.addEventListener('click', function() {
        var text = code.textContent;
        navigator.clipboard.writeText(text).then(function() {
          btn.innerHTML = ICONS.check;
          btn.classList.add('copied');
          setTimeout(function() {
            btn.innerHTML = ICONS.copy;
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
      var icon = ICONS[type] || '';
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
