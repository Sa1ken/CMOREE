(() => {
  const ready = () => {
    const body = document.body;
    if (!body) return;

    const prefersReduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const syncBrandNames = () => {
      document.querySelectorAll(".brand-name").forEach((node) => {
        node.style.display = "inline-block";
        node.style.background = "none";
        node.style.color = "#dff4ff";
        node.style.webkitTextFillColor = "#dff4ff";
      });
    };

    syncBrandNames();

    if (!document.querySelector(".psx-ambient")) {
      const ambient = document.createElement("div");
      ambient.className = "psx-ambient";
      ambient.setAttribute("aria-hidden", "true");
      ambient.innerHTML = [
        '<div class="psx-orb a"></div>',
        '<div class="psx-orb b"></div>',
        '<div class="psx-orb c"></div>',
        '<div class="psx-ribbon a"></div>',
        '<div class="psx-ribbon b"></div>',
        '<div class="psx-ribbon c"></div>',
        '<div class="psx-cursor"></div>',
        '<div class="psx-vignette"></div>'
      ].join("");
      body.prepend(ambient);
    }

    const revealSelector = [
      ".topbar",
      ".sidebar",
      ".page-header",
      ".hero",
      ".form-card",
      ".stat-card",
      ".panel",
      ".maint-launch-card",
      ".mini-card",
      ".feature",
      ".db-file-item",
      ".db-table-item",
      ".sched-item",
      ".audit-item",
      ".log-pack-item",
      ".log-file-item",
      ".user-row"
    ].join(",");

    const tiltSelector = [
      ".hero",
      ".form-card",
      ".maint-modal-dialog",
      ".action-confirm-dialog",
      ".page-header"
    ].join(",");

    const decorate = (root = document) => {
      const motionNodes = root.querySelectorAll(revealSelector);
      motionNodes.forEach((node, index) => {
        node.classList.add("motion-item");
        node.style.setProperty("--motion-order", String(index % 12));
      });

      const tiltNodes = root.querySelectorAll(tiltSelector);
      tiltNodes.forEach((node) => node.classList.add("tilt-surface"));
    };

    decorate();

    const revealObserver = prefersReduced
      ? null
      : new IntersectionObserver(
          (entries) => {
            entries.forEach((entry) => {
              if (!entry.isIntersecting) return;
              entry.target.classList.add("is-visible");
              revealObserver.unobserve(entry.target);
            });
          },
          {
            threshold: 0.16,
            rootMargin: "0px 0px -6% 0px"
          }
        );

    const observeMotion = (root = document) => {
      root.querySelectorAll(".motion-item").forEach((node) => {
        if (prefersReduced) {
          node.classList.add("is-visible");
          return;
        }
        if (node.classList.contains("is-visible")) return;
        revealObserver.observe(node);
      });
    };

    observeMotion();

    const showInitialMotion = (root = document) => {
      root.querySelectorAll(".motion-item").forEach((node) => {
        const rect = node.getBoundingClientRect();
        const visible = rect.width > 0 && rect.height > 0 && rect.bottom > 0 && rect.top < window.innerHeight * 0.96;
        if (visible) {
          node.classList.add("is-visible");
        }
      });
    };

    showInitialMotion();

    const replayVisiblePage = (page) => {
      if (!page || prefersReduced) return;
      const items = page.querySelectorAll(".motion-item");
      items.forEach((node, index) => {
        node.classList.remove("is-visible");
        node.style.setProperty("--motion-order", String(index % 12));
      });
      requestAnimationFrame(() => {
        items.forEach((node) => node.classList.add("is-visible"));
      });
    };

    const pageObserver = new MutationObserver((entries) => {
      entries.forEach((entry) => {
        const target = entry.target;
        if (!(target instanceof HTMLElement)) return;
        if (!target.classList.contains("page")) return;
        if (!target.classList.contains("on")) return;
        decorate(target);
        observeMotion(target);
        showInitialMotion(target);
        bindTilt(target);
        replayVisiblePage(target);
      });
    });

    document.querySelectorAll(".page").forEach((page) => {
      pageObserver.observe(page, { attributes: true, attributeFilter: ["class"] });
    });

    if (!prefersReduced) {
      let pointerFrame = 0;
      window.addEventListener(
        "pointermove",
        (event) => {
          if (event.pointerType === "touch") return;
          if (pointerFrame) cancelAnimationFrame(pointerFrame);
          pointerFrame = requestAnimationFrame(() => {
            const x = (event.clientX / window.innerWidth) * 100;
            const y = (event.clientY / window.innerHeight) * 100;
            body.style.setProperty("--fx-pointer-x", `${x.toFixed(2)}%`);
            body.style.setProperty("--fx-pointer-y", `${y.toFixed(2)}%`);
          });
        },
        { passive: true }
      );
    }

    const supportsTilt = window.matchMedia("(hover: hover) and (pointer: fine)").matches && !prefersReduced;

    const bindTilt = (root = document) => {
      if (!supportsTilt) return;
      root.querySelectorAll(".tilt-surface").forEach((node) => {
        if (node.dataset.tiltBound === "1") return;
        node.dataset.tiltBound = "1";

        const reset = () => {
          node.style.setProperty("--tilt-x", "0deg");
          node.style.setProperty("--tilt-y", "0deg");
          node.style.setProperty("--sheen-x", "50%");
          node.style.setProperty("--sheen-y", "50%");
        };

        node.addEventListener("pointermove", (event) => {
          if (event.pointerType === "touch") return;
          const rect = node.getBoundingClientRect();
          const px = (event.clientX - rect.left) / rect.width - 0.5;
          const py = (event.clientY - rect.top) / rect.height - 0.5;
          node.style.setProperty("--tilt-x", `${(px * 5).toFixed(2)}deg`);
          node.style.setProperty("--tilt-y", `${(-py * 5).toFixed(2)}deg`);
          node.style.setProperty("--sheen-x", `${((px + 0.5) * 100).toFixed(1)}%`);
          node.style.setProperty("--sheen-y", `${((py + 0.5) * 100).toFixed(1)}%`);
        });

        node.addEventListener("pointerleave", reset);
        node.addEventListener("pointercancel", reset);
        reset();
      });
    };

    bindTilt();

    requestAnimationFrame(() => {
      body.classList.add("fx-ready");
      document.querySelectorAll(".motion-item").forEach((node) => {
        if (prefersReduced) {
          node.classList.add("is-visible");
        }
      });
    });
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", ready, { once: true });
  } else {
    ready();
  }
})();
