// TennisPro Shared JS — cart, search, navigation, interactions
(function() {
  'use strict';

  // ─── Mock Data ───
  var CATEGORIES = [
    {id:'rackets', name:'网球拍', icon:'🎾', desc:'专业网球拍，碳纤维轻量化'},
    {id:'balls', name:'网球/球包', icon:'🥎', desc:'比赛训练用球，多功能球包'},
    {id:'shoes', name:'网球鞋', icon:'👟', desc:'专业网球运动鞋，防滑耐磨'},
    {id:'apparel', name:'网球服装', icon:'👕', desc:'透气速干运动服装'},
    {id:'bags', name:'球包/护具', icon:'🎒', desc:'专业球包与运动护具'},
    {id:'accessories', name:'配件', icon:'⌚', desc:'手胶、避震器、护腕等'}
  ];

  var PRODUCTS = [
    {id:1,name:'Babolat Pure Aero 网球拍',cat:'rackets',price:1899,originalPrice:2299,sales:1280,stock:15,tag:'热销',imgs:3},
    {id:2,name:'Wilson Pro Staff RF97 网球拍',cat:'rackets',price:2299,originalPrice:2699,sales:860,stock:8,tag:'新品',imgs:3},
    {id:3,name:'Head Speed Pro 网球拍',cat:'rackets',price:1699,originalPrice:1999,sales:620,stock:0,imgs:3},
    {id:4,name:'Dunlop AO 网球(4球装)',cat:'balls',price:89,originalPrice:120,sales:5200,stock:200,imgs:2},
    {id:5,name:'Wilson US Open 网球(4球装)',cat:'balls',price:79,originalPrice:99,sales:3800,stock:150,imgs:2},
    {id:6,name:'Nike Air Zoom Vapor 网球鞋',cat:'shoes',price:1099,originalPrice:1299,sales:920,stock:25,imgs:3},
    {id:7,name:'Adidas Barricade 网球鞋',cat:'shoes',price:899,originalPrice:1099,sales:680,stock:18,imgs:3},
    {id:8,name:'Fila 网球Polo衫 男款',cat:'apparel',price:299,originalPrice:399,sales:1500,stock:80,imgs:2},
    {id:9,name:'Nike 网球短裤 速干款',cat:'apparel',price:199,originalPrice:259,sales:1100,stock:60,imgs:2},
    {id:10,name:'Babolat 12支装网球包',cat:'bags',price:599,originalPrice:799,sales:420,stock:12,imgs:3},
    {id:11,name:'Yonex 运动护腕(2只装)',cat:'bags',price:69,originalPrice:89,sales:2800,stock:100,imgs:2},
    {id:12,name:'Wilson 网球手胶(3条装)',cat:'accessories',price:59,originalPrice:79,sales:4500,stock:300,imgs:2},
    {id:13,name:'Tourna 避震器套装',cat:'accessories',price:39,originalPrice:49,sales:3200,stock:250,imgs:2},
    {id:14,name:'Head 网球帽 遮阳款',cat:'apparel',price:99,originalPrice:139,sales:800,stock:45,imgs:2}
  ];

  // ─── User (localStorage) ───
  function getUser() {
    try { return JSON.parse(localStorage.getItem('tennispro_user') || 'null'); }
    catch(e) { return null; }
  }
  function saveUser(user) {
    localStorage.setItem('tennispro_user', JSON.stringify(user));
  }
  function clearUser() {
    localStorage.removeItem('tennispro_user');
  }
  function isLoggedIn() {
    return !!getUser();
  }
  function requireLogin(returnUrl) {
    if (!isLoggedIn()) {
      var url = returnUrl || window.location.href;
      window.location.href = 'login.html?returnUrl=' + encodeURIComponent(url);
      return false;
    }
    return true;
  }

  // ─── Addresses (localStorage) ───
  function getAddresses() {
    try { return JSON.parse(localStorage.getItem('tennispro_addresses') || '[]'); }
    catch(e) { return []; }
  }
  function saveAddresses(addrs) {
    localStorage.setItem('tennispro_addresses', JSON.stringify(addrs));
  }

  // ─── Orders (localStorage) ───
  function getOrders() {
    try { return JSON.parse(localStorage.getItem('tennispro_orders') || '[]'); }
    catch(e) { return []; }
  }
  function saveOrders(orders) {
    localStorage.setItem('tennispro_orders', JSON.stringify(orders));
  }
  function addOrder(order) {
    var orders = getOrders();
    orders.unshift(order);
    saveOrders(orders);
  }

  // ─── Cart (localStorage) ───
  function getCart() {
    try { return JSON.parse(localStorage.getItem('tennispro_cart') || '[]'); }
    catch(e) { return []; }
  }
  function saveCart(cart) {
    localStorage.setItem('tennispro_cart', JSON.stringify(cart));
    updateCartBadge();
  }
  function addToCart(productId, qty, spec) {
    var cart = getCart();
    var p = PRODUCTS.find(function(x){return x.id===productId;});
    if (!p) return;
    var key = productId + '_' + (spec || 'default');
    var existing = cart.find(function(x){return x.key===key;});
    if (existing) { existing.qty += qty; }
    else {
      cart.push({key:key, productId:productId, name:p.name, price:p.price, qty:qty, spec:spec||'标准', img:p.cat, selected:true});
    }
    saveCart(cart);
    showToast('已加入购物车');
  }
  function removeFromCart(key) {
    var cart = getCart().filter(function(x){return x.key!==key;});
    saveCart(cart);
  }
  function updateQty(key, delta) {
    var cart = getCart();
    var item = cart.find(function(x){return x.key===key;});
    if (item) { item.qty = Math.max(1, item.qty + delta); saveCart(cart); }
  }
  function toggleSelect(key, selected) {
    var cart = getCart();
    var item = cart.find(function(x){return x.key===key;});
    if (item) { item.selected = selected; saveCart(cart); }
  }
  function getSelectedCart() {
    return getCart().filter(function(x){return x.selected !== false;});
  }
  function clearCart() {
    localStorage.setItem('tennispro_cart', '[]');
    updateCartBadge();
  }
  function updateCartBadge() {
    var cart = getCart();
    var count = cart.reduce(function(s,x){return s+x.qty;},0);
    document.querySelectorAll('.cart-badge').forEach(function(el){
      el.textContent = count;
      el.style.display = count > 0 ? 'flex' : 'none';
    });
  }

  // ─── Toast ───
  function showToast(msg) {
    var existing = document.querySelector('.toast');
    if (existing) existing.remove();
    var toast = document.createElement('div');
    toast.className = 'toast';
    toast.innerHTML = '<span class="toast-icon">✓</span><span>'+msg+'</span>';
    document.body.appendChild(toast);
    requestAnimationFrame(function(){ toast.classList.add('show'); });
    setTimeout(function(){ toast.classList.remove('show'); setTimeout(function(){toast.remove();},300); }, 2500);
  }

  // ─── Search ───
  function handleSearch(query) {
    if (query && query.trim()) {
      window.location.href = 'search.html?q=' + encodeURIComponent(query.trim());
    }
  }

  // ─── Order number generator ───
  function generateOrderNo() {
    var now = new Date();
    var ts = now.getFullYear().toString().slice(2) +
      (now.getMonth()+1).toString().padStart(2,'0') +
      now.getDate().toString().padStart(2,'0') +
      now.getHours().toString().padStart(2,'0') +
      now.getMinutes().toString().padStart(2,'0') +
      now.getSeconds().toString().padStart(2,'0');
    var rand = Math.floor(Math.random() * 1000).toString().padStart(3,'0');
    return 'TP' + ts + rand;
  }

  // ─── Nav HTML ───
  function navHTML(active) {
    var user = getUser();
    var userIcon = user ? '👤' : '👤';
    var userHref = user ? 'account.html' : 'login.html';
    return '\
    <nav class="navbar">\
      <div class="navbar-inner">\
        <a href="index.html" class="nav-logo">Tennis<span>Pro</span></a>\
        <div class="nav-links">\
          <a href="index.html"' +(active==='home'?' class="active"':'')+ '>首页</a>\
          <a href="products.html"' +(active==='products'?' class="active"':'')+ '>全部商品</a>\
          <a href="products.html?cat=rackets">网球拍</a>\
          <a href="products.html?cat=shoes">网球鞋</a>\
          <a href="products.html?cat=apparel">服装</a>\
        </div>\
        <div class="nav-right">\
          <div class="nav-search">\
            <span style="opacity:.5;margin-right:8px;">🔍</span>\
            <input type="text" id="navSearch" placeholder="搜索网球装备..." />\
          </div>\
          <a href="' + userHref + '" class="nav-icon-btn" title="' + (user ? user.name : '登录/注册') + '">' + userIcon + '</a>\
          <a href="cart.html" class="nav-icon-btn" title="购物车">🛒\
            <span class="cart-badge">0</span>\
          </a>\
        </div>\
      </div>\
    </nav>';
  }

  // ─── Footer HTML ───
  function footerHTML() {
    return '\
    <footer class="footer">\
      <div class="container">\
        <div class="footer-grid">\
          <div class="footer-col">\
            <div class="footer-brand">Tennis<span>Pro</span></div>\
            <p style="color:var(--muted);font-size:var(--fs-small);line-height:1.6;max-width:320px;">你的网球装备，一站搞定。专业网球运动品牌在线商城，提供从球拍到配件的全品类网球商品。</p>\
          </div>\
          <div class="footer-col">\
            <h4>购物指南</h4>\
            <a href="#">购物流程</a><a href="#">支付方式</a><a href="#">配送说明</a><a href="#">运费规则</a>\
          </div>\
          <div class="footer-col">\
            <h4>客户服务</h4>\
            <a href="#">常见问题</a><a href="#">联系客服</a><a href="#">意见反馈</a><a href="#">关于我们</a>\
          </div>\
          <div class="footer-col">\
            <h4>关注我们</h4>\
            <a href="#">微信公众号</a><a href="#">微博</a><a href="#">小红书</a><a href="#">抖音</a>\
          </div>\
        </div>\
        <div class="footer-bottom">\
          <span>© 2026 TennisPro. All rights reserved.</span>\
          <span>满99元免运费 | 支付宝/微信支付</span>\
        </div>\
      </div>\
    </footer>';
  }

  // ─── Init ───
  document.addEventListener('DOMContentLoaded', function() {
    // Inject nav if placeholder exists
    var navPlaceholder = document.getElementById('nav-placeholder');
    if (navPlaceholder) {
      var active = navPlaceholder.getAttribute('data-active') || '';
      navPlaceholder.outerHTML = navHTML(active);
    }
    // Inject footer if placeholder exists
    var footerPlaceholder = document.getElementById('footer-placeholder');
    if (footerPlaceholder) {
      footerPlaceholder.outerHTML = footerHTML();
    }
    // Cart badge
    updateCartBadge();
    // Search handler
    var navSearch = document.getElementById('navSearch');
    if (navSearch) {
      navSearch.addEventListener('keypress', function(e){
        if (e.key === 'Enter') handleSearch(navSearch.value);
      });
      // Pre-fill from URL
      var params = new URLSearchParams(window.location.search);
      if (params.get('q')) navSearch.value = params.get('q');
    }
  });

  // ─── Export ───
  window.TennisPro = {
    CATEGORIES: CATEGORIES,
    PRODUCTS: PRODUCTS,
    getCart: getCart,
    saveCart: saveCart,
    addToCart: addToCart,
    removeFromCart: removeFromCart,
    updateQty: updateQty,
    toggleSelect: toggleSelect,
    getSelectedCart: getSelectedCart,
    clearCart: clearCart,
    updateCartBadge: updateCartBadge,
    showToast: showToast,
    handleSearch: handleSearch,
    getUser: getUser,
    saveUser: saveUser,
    clearUser: clearUser,
    isLoggedIn: isLoggedIn,
    requireLogin: requireLogin,
    getAddresses: getAddresses,
    saveAddresses: saveAddresses,
    getOrders: getOrders,
    saveOrders: saveOrders,
    addOrder: addOrder,
    generateOrderNo: generateOrderNo
  };
})();
