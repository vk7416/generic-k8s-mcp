(function () {
  const root = document.documentElement;
  const toggle = document.getElementById("theme-toggle");
  const storageKey = "generic-k8s-mcp-theme";

  function resolveTheme(savedTheme) {
    if (savedTheme === "light" || savedTheme === "dark") {
      return savedTheme;
    }

    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }

  function applyTheme(theme) {
    root.setAttribute("data-theme", theme);
    if (toggle) {
      toggle.setAttribute("aria-pressed", String(theme === "dark"));
    }
  }

  const initialTheme = resolveTheme(localStorage.getItem(storageKey));
  applyTheme(initialTheme);

  if (toggle) {
    toggle.addEventListener("click", function () {
      const nextTheme = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
      localStorage.setItem(storageKey, nextTheme);
      applyTheme(nextTheme);
    });
  }

  const media = window.matchMedia("(prefers-color-scheme: dark)");
  media.addEventListener("change", function () {
    if (!localStorage.getItem(storageKey)) {
      applyTheme(resolveTheme(null));
    }
  });
})();
