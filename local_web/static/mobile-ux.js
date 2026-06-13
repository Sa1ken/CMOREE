(function(){
  const mq = window.matchMedia('(max-width: 820px)');
  const primaryTabs = [
    { id: 'dash', label: 'Панель', icon: 'fa-gauge-high' },
    { id: 'console', label: 'Консоль', icon: 'fa-terminal' },
    { id: 'configs', label: 'Конфиги', icon: 'fa-sliders' },
    { id: 'files', label: 'Файлы', icon: 'fa-folder-tree' },
    { id: 'mods', label: 'Аддоны', icon: 'fa-puzzle-piece' },
    { id: 'more', label: 'Ещё', icon: 'fa-bars' }
  ];

  function isDashboardPage(){
    return !!document.querySelector('.layout .sidebar .nav-item');
  }

  function currentPageId(){
    const active = document.querySelector('.page.on[id^="page-"]');
    if (!active) return 'dash';
    return active.id.replace(/^page-/, '');
  }

  function clickNav(pageId){
    const esc = window.CSS && CSS.escape ? CSS.escape(pageId) : String(pageId).replace(/\"/g, '');
    const item = document.querySelector(`.sidebar .nav-item[data-p="${esc}"]`);
    if (!item || item.hidden || getComputedStyle(item).display === 'none' || getComputedStyle(item).visibility === 'hidden') {
      openDrawer();
      return;
    }
    item.click();
  }

  function syncTabs(){
    const active = currentPageId();
    document.querySelectorAll('.mobile-tab').forEach(btn => {
      const id = btn.dataset.mobilePage;
      btn.classList.toggle('is-active', id !== 'more' && id === active);
    });
  }

  function closeDrawer(){
    document.body.classList.remove('mobile-nav-open');
    if (mq.matches && typeof window.applySidebarState === 'function') {
      window.applySidebarState(true);
    }
  }

  function openDrawer(){
    if (typeof window.applySidebarState === 'function') window.applySidebarState(false);
    document.body.classList.add('mobile-nav-open');
    const search = document.getElementById('navSearchInput');
    if (search) setTimeout(() => search.focus({ preventScroll: true }), 80);
  }

  function toggleDrawer(){
    if (document.body.classList.contains('mobile-nav-open')) closeDrawer();
    else openDrawer();
  }

  function ensureShell(){
    if (!isDashboardPage()) return;
    if (!document.querySelector('.mobile-sidebar-backdrop')) {
      const backdrop = document.createElement('button');
      backdrop.type = 'button';
      backdrop.className = 'mobile-sidebar-backdrop';
      backdrop.setAttribute('aria-label', 'Закрыть меню');
      backdrop.addEventListener('click', closeDrawer);
      document.body.appendChild(backdrop);
    }
    if (!document.querySelector('.mobile-tabbar')) {
      const nav = document.createElement('nav');
      nav.className = 'mobile-tabbar';
      nav.setAttribute('aria-label', 'Быстрая навигация');
      nav.innerHTML = primaryTabs.map(tab => `
        <button class="mobile-tab" type="button" data-mobile-page="${tab.id}" aria-label="${tab.label}">
          <i class="fas ${tab.icon}"></i><span>${tab.label}</span>
        </button>`).join('');
      nav.addEventListener('click', event => {
        const btn = event.target.closest('.mobile-tab');
        if (!btn) return;
        const id = btn.dataset.mobilePage;
        if (id === 'more') toggleDrawer();
        else {
          clickNav(id);
          closeDrawer();
          setTimeout(syncTabs, 0);
        }
      });
      document.body.appendChild(nav);
    }
    syncTabs();
  }

  function patchSidebarToggle(){
    if (!isDashboardPage() || window.__mobileUxTogglePatched) return;
    window.__mobileUxTogglePatched = true;
    const nativeToggle = window.toggleSidebar;
    window.toggleSidebar = function(forceHidden = null){
      if (mq.matches) {
        const shouldOpen = typeof forceHidden === 'boolean'
          ? !forceHidden
          : !document.body.classList.contains('mobile-nav-open');
        if (shouldOpen) openDrawer();
        else closeDrawer();
        return;
      }
      document.body.classList.remove('mobile-nav-open');
      if (typeof nativeToggle === 'function') return nativeToggle(forceHidden);
    };
  }

  function applyMode(){
    ensureShell();
    patchSidebarToggle();
    if (!isDashboardPage()) return;
    if (mq.matches) closeDrawer();
    else document.body.classList.remove('mobile-nav-open');
    syncTabs();
  }

  document.addEventListener('click', event => {
    const navItem = event.target.closest('.sidebar .nav-item');
    if (!navItem) return;
    setTimeout(syncTabs, 0);
    if (mq.matches) setTimeout(closeDrawer, 0);
  });

  document.addEventListener('keydown', event => {
    if (event.key === 'Escape' && document.body.classList.contains('mobile-nav-open')) closeDrawer();
  });

  const observer = new MutationObserver(syncTabs);
  document.addEventListener('DOMContentLoaded', () => {
    applyMode();
    const main = document.querySelector('.main');
    if (main) observer.observe(main, { subtree: true, attributes: true, attributeFilter: ['class'] });
  });
  if (document.readyState !== 'loading') applyMode();
  if (mq.addEventListener) mq.addEventListener('change', applyMode);
  else mq.addListener(applyMode);
})();
