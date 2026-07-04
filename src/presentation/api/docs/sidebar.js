(function () {
  var origOnload = window.onload;
  if (origOnload) {
    window.onload = function () {
      var OriginalBundle = window.SwaggerUIBundle;
      var wrapped = function (opts) {
        if (opts.spec && opts.spec.paths) {
          var pathOrder = Object.keys(opts.spec.paths);
          opts.operationsSorter = function (a, b) {
            return pathOrder.indexOf(a.get("path")) - pathOrder.indexOf(b.get("path"));
          };
        }
        return OriginalBundle(opts);
      };
      Object.assign(wrapped, OriginalBundle);
      window.SwaggerUIBundle = wrapped;
      origOnload();
      window.SwaggerUIBundle = OriginalBundle;
    };
  }

  const METHODS = ["get", "post", "put", "patch", "delete", "head", "options"];

  function waitForSwagger() {
    const ui = document.querySelector(".swagger-ui");
    if (ui && !document.getElementById("api-sidebar")) {
      initSidebar();
      return;
    }
    setTimeout(waitForSwagger, 200);
  }

  async function initSidebar() {
    const res = await fetch("/api/openapi.json");
    const spec = await res.json();
    buildSidebar(spec);
    document.body.classList.add("swagger-sidebar-enabled");
    ensureTagSectionsOpen();
    syncActiveLink();
    window.addEventListener("hashchange", syncActiveLink);
  }

  function ensureTagSectionsOpen() {
    document.querySelectorAll(".swagger-ui .opblock-tag").forEach(function (tag) {
      var section = tag.parentElement;
      if (!section) return;
      if (!section.classList.contains("is-open")) {
        tag.click();
      }
    });
  }

  function closeAllGroups(nav) {
    nav.querySelectorAll(".sidebar-group").forEach(function (group) {
      group.classList.remove("open");
    });
  }

  function openGroupForHash(nav) {
    const hash = window.location.hash;
    if (!hash) return;
    nav.querySelectorAll(".sidebar-link").forEach(function (link) {
      if (link.getAttribute("href") === hash) {
        link.closest(".sidebar-group")?.classList.add("open");
      }
    });
  }

  function buildSidebar(spec) {
    const sidebar = document.createElement("nav");
    sidebar.id = "api-sidebar";

    const header = document.createElement("div");
    header.className = "sidebar-header";
    header.innerHTML =
      "<h1>" + (spec.info?.title || "API Docs") + "</h1>" +
      "<p>v" + (spec.info?.version || "1.0.0") + "</p>";
    sidebar.appendChild(header);

    const searchWrap = document.createElement("div");
    searchWrap.className = "sidebar-search";
    const search = document.createElement("input");
    search.type = "search";
    search.placeholder = "Filter routes...";
    searchWrap.appendChild(search);
    sidebar.appendChild(searchWrap);

    const nav = document.createElement("div");
    nav.className = "sidebar-nav";

    const tagOrder = (spec.tags || []).map(function (t) { return t.name; });
    const pathOrder = Object.keys(spec.paths || {});
    const grouped = {};

    for (const path of pathOrder) {
      const item = spec.paths[path];
      for (const method of METHODS) {
        if (!item[method]) continue;
        const op = item[method];
        const tag = (op.tags && op.tags[0]) || "Other";
        if (!grouped[tag]) grouped[tag] = [];
        grouped[tag].push({
          method: method,
          path: path,
          summary: op.summary || path,
          operationId: op.operationId || method + path.replace(/[{}\/]/g, "_"),
        });
      }
    }

    const tags = tagOrder.filter(function (t) { return grouped[t]; });
    Object.keys(grouped).forEach(function (t) {
      if (tags.indexOf(t) === -1) tags.push(t);
    });

    tags.forEach(function (tag) {
      const group = document.createElement("div");
      group.className = "sidebar-group";
      group.dataset.tag = tag;

      const title = document.createElement("button");
      title.type = "button";
      title.className = "sidebar-group-title";
      title.innerHTML =
        '<span class="chevron">▶</span>' +
        "<span>" + tag + "</span>" +
        '<span class="sidebar-group-count">' + grouped[tag].length + "</span>";
      title.addEventListener("click", function () {
        const isOpen = group.classList.contains("open");
        closeAllGroups(nav);
        if (!isOpen) group.classList.add("open");
      });
      group.appendChild(title);

      const items = document.createElement("div");
      items.className = "sidebar-group-items";

      grouped[tag].sort(function (a, b) {
        return pathOrder.indexOf(a.path) - pathOrder.indexOf(b.path);
      });

      grouped[tag].forEach(function (op) {
        const link = document.createElement("a");
        link.className = "sidebar-link";
        link.href = "#/" + encodeURIComponent(tag) + "/" + op.operationId;
        link.dataset.operationId = op.operationId;
        link.innerHTML =
          '<span class="method method-' + op.method + '">' + op.method + "</span>" +
          '<span class="sidebar-link-text">' + op.summary + "</span>";
        link.addEventListener("click", function () {
          closeAllGroups(nav);
          group.classList.add("open");
          setTimeout(function () {
            var target = document.querySelector(
              '.swagger-ui .opblock[data-operation-id="' + op.operationId + '"]',
            );
            if (target) {
              target.scrollIntoView({ behavior: "smooth", block: "start" });
            }
          }, 100);
        });
        items.appendChild(link);
      });

      group.appendChild(items);
      nav.appendChild(group);
    });

    sidebar.appendChild(nav);
    document.body.insertBefore(sidebar, document.body.firstChild);

    search.addEventListener("input", function () {
      const q = search.value.toLowerCase().trim();
      if (!q) {
        closeAllGroups(nav);
        openGroupForHash(nav);
        nav.querySelectorAll(".sidebar-group").forEach(function (group) {
          group.style.display = "block";
          group.querySelectorAll(".sidebar-link").forEach(function (link) {
            link.style.display = "flex";
          });
        });
        return;
      }

      nav.querySelectorAll(".sidebar-group").forEach(function (group) {
        let visible = 0;
        group.querySelectorAll(".sidebar-link").forEach(function (link) {
          const text = link.textContent.toLowerCase();
          const show = text.indexOf(q) !== -1;
          link.style.display = show ? "flex" : "none";
          if (show) visible++;
        });
        group.style.display = visible ? "block" : "none";
        group.classList.toggle("open", visible > 0);
      });
    });
  }

  function syncActiveLink() {
    const hash = window.location.hash;
    document.querySelectorAll("#api-sidebar .sidebar-link").forEach(function (link) {
      link.classList.toggle("active", hash && link.getAttribute("href") === hash);
    });

    const nav = document.querySelector("#api-sidebar .sidebar-nav");
    if (!nav || !hash) return;
    closeAllGroups(nav);
    openGroupForHash(nav);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", waitForSwagger);
  } else {
    waitForSwagger();
  }
})();
