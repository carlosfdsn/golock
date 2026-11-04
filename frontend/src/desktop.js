// GoLock · extras de desktop. Usa o DOM já existente (clica nos mesmos botões),
// sem alterar a lógica do main.js.
import "./desktop.css";
import { CopyToClipboard, ListEntries } from "../wailsjs/go/main/App";

const $ = (id) => document.getElementById(id);
const desktop = () => matchMedia("(min-width: 1000px) and (min-height: 560px)").matches;
const cofreAberto = () => !$("tela-cofre").classList.contains("escondida");
const modalAberto = () => [...document.querySelectorAll(".modal")].some((m) => !m.classList.contains("escondida"));
const cards = () => [...document.querySelectorAll("#lista .card")];
const chave = (c) => c.querySelector("strong").textContent + "\u0000" + c.querySelector(".user").textContent;

const I = {
  copy: '<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1"/>',
  edit: '<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/>',
  trash: '<path d="M3 6h18M8 6V4h8v2M6 6l1 14h10l1-14M10 11v5M14 11v5"/>',
  eye: '<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
};
const sv = (n) => `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${I[n]}</svg>`;
function forca(s) {
  if (!s) return 0;
  let p = 0;
  if (s.length >= 8) p++;
  if (s.length >= 12) p++;
  if (/[a-z]/.test(s) && /[A-Z]/.test(s)) p++;
  if (/\d/.test(s)) p++;
  if (/[^A-Za-z0-9]/.test(s)) p++;
  return p <= 1 ? 1 : p === 2 ? 2 : p <= 4 ? 3 : 4;
}
const ROT = ["", "Fraca", "Razoável", "Boa", "Forte"];

// Estrutura extra: painel de detalhe, botão "+" na barra, barra de atalhos
const cofre = $("tela-cofre");
cofre.insertAdjacentHTML("beforeend", `<aside id="detalhe" class="detalhe"></aside>
  <footer class="atalhos" aria-hidden="true">
    <span><kbd>Ctrl+N</kbd>nova</span><span><kbd>Ctrl+K</kbd>buscar</span><span><kbd>↑</kbd><kbd>↓</kbd>navegar</span>
    <span><kbd>Enter</kbd>copiar senha</span><span><kbd>U</kbd>usuário</span><span><kbd>E</kbd>editar</span>
    <span><kbd>Del</kbd>apagar</span><span><kbd>Ctrl+L</kbd>travar</span><span class="espaco"></span><span id="st-clip"></span><span id="st-trava"></span>
  </footer>`);
const novoD = document.createElement("button");
novoD.id = "btn-novo-d"; novoD.className = "icon-btn ouro"; novoD.title = "Nova senha (Ctrl+N)";
novoD.setAttribute("aria-label", "Nova senha"); novoD.innerHTML = sv("plus");
novoD.onclick = () => $("btn-novo").click();
document.querySelector(".topo .acoes").prepend(novoD);

// Busca no centro da barra superior (volta ao lugar original em telas pequenas)
const busca = document.querySelector("label.busca");
const marcador = document.createComment("busca");
busca.before(marcador);
const kbdBusca = document.createElement("kbd");
kbdBusca.className = "busca-kbd"; kbdBusca.textContent = "Ctrl K";
busca.append(kbdBusca);
const posBusca = () => (desktop() ? document.querySelector(".topo .acoes").before(busca) : marcador.after(busca));
posBusca();
matchMedia("(min-width: 1000px) and (min-height: 560px)").addEventListener("change", posBusca);

// Filtro "para melhorar": clicar no resumo mostra só as senhas fracas
document.querySelector(".stat.aviso").addEventListener("click", (ev) => {
  if (!desktop()) return;
  ev.currentTarget.classList.toggle("on", $("lista").classList.toggle("so-fracas"));
});

// Bloqueio automático: contagem na barra inferior
let ultima = Date.now();
["pointerdown", "pointermove", "keydown", "wheel"].forEach((ev) =>
  addEventListener(ev, () => (ultima = Date.now()), { passive: true, capture: true }));
setInterval(() => {
  if (!cofreAberto()) return;
  let min = 5;
  try { min = Number(localStorage.getItem("autolock")) || 5; } catch {}
  const s = Math.max(0, Math.ceil((min * 60000 - (Date.now() - ultima)) / 1000));
  $("st-trava").textContent = `Trava em ${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}, 1000);

let tAviso;
function aviso(msg) {
  $("toast-txt").textContent = msg;
  $("toast").classList.add("mostra");
  clearTimeout(tAviso);
  tAviso = setTimeout(() => $("toast").classList.remove("mostra"), 2000);
}

const itemMenu = (re) => [...document.querySelectorAll("#menu-card button")].find((b) => re.test(b.textContent));
const doMenu = (card, re) => { card.querySelector(".mais").click(); itemMenu(re)?.click(); };

// ---- Painel de detalhe ----
let foco = null, ver = false, tVer, seq = 0;
const vazio = () => `<div class="det-vazio"><img src="${document.querySelector("img.mascote")?.src || ""}" alt=""><strong>Nenhuma senha selecionada</strong><span>Escolha uma na lista ou use ↑ ↓.</span></div>`;

async function atualizarDetalhe() {
  const box = $("detalhe"), n = ++seq;
  const card = cards().find((c) => chave(c) === foco);
  if (!card) { box.innerHTML = vazio(); return; }
  let e;
  try { e = ((await ListEntries()) || []).find((x) => x.name + "\u0000" + (x.username || "Sem usuário") === foco); } catch {}
  if (n !== seq) return;
  if (!e) { box.innerHTML = vazio(); return; }
  const f = forca(e.password);
  box.innerHTML = `<div class="det-int">
    <div class="det-cab"><div class="avatar">${card.querySelector(".avatar").innerHTML}</div><h2></h2><span></span></div>
    <div class="det-acoes">
      <button data-a="senha"><i>${sv("copy")}</i>Copiar senha</button>
      <button data-a="editar"><i>${sv("edit")}</i>Editar</button>
      <button data-a="apagar" class="perigo"><i>${sv("trash")}</i>Apagar</button>
    </div>
    <div class="det-grupo">
      <div class="det-linha"><small>Usuário</small><span class="u"></span><button data-a="user" aria-label="Copiar usuário">${sv("copy")}</button></div>
      <div class="det-linha"><small>Senha</small><span class="mono p"></span><button data-a="ver" aria-label="Mostrar senha">${sv("eye")}</button><button data-a="senha" aria-label="Copiar senha">${sv("copy")}</button></div>
      <div class="det-linha"><small>Força</small><div class="forca" data-n="${f}"><i></i><i></i><i></i><i></i><small>${ROT[f]}</small></div></div>
    </div></div>`;
  // textContent: seguro contra HTML digitado pelo usuário
  box.querySelector("h2").textContent = e.name;
  box.querySelector(".det-cab span").textContent = card.closest(".grupo")?.querySelector(".grupo-titulo span")?.textContent || "";
  box.querySelector(".u").textContent = e.username || "Sem usuário";
  box.querySelector(".p").textContent = ver ? e.password : "••••••••••";
  if (f <= 2) box.querySelector(".det-int").insertAdjacentHTML("beforeend",
    '<div class="aviso-box"><span>⚠</span><p>Esta senha é <b>fraca</b>. Edite e use o gerador para criar uma forte.</p></div>');
  box.onclick = (ev) => {
    const b = ev.target.closest("button[data-a]");
    if (!b) return;
    const a = b.dataset.a;
    if (a === "senha") card.querySelector(".copiar").click();
    else if (a === "editar") doMenu(card, /Editar/);
    else if (a === "apagar") doMenu(card, /Apagar/);
    else if (a === "user") e.username ? CopyToClipboard(e.username).then(() => aviso("Usuário copiado · some em 15s")) : aviso("Esta entrada não tem usuário");
    else if (a === "ver") {
      ver = !ver; clearTimeout(tVer);
      if (ver) tVer = setTimeout(() => { ver = false; atualizarDetalhe(); }, 10000); // oculta sozinha após 10s
      atualizarDetalhe();
    }
  };
}

function aplicarFoco(rolar, mudou) {
  cards().forEach((c) => {
    const on = chave(c) === foco;
    c.classList.toggle("foco", on);
    if (on && rolar) c.scrollIntoView({ block: "nearest" });
  });
  if (mudou) { ver = false; clearTimeout(tVer); }
  if (desktop()) atualizarDetalhe();
}
const focar = (c, rolar) => { foco = chave(c); aplicarFoco(rolar, true); };
new MutationObserver(() => aplicarFoco(false, false)).observe($("lista"), { childList: true });

$("lista").addEventListener("click", (ev) => { const c = ev.target.closest(".card"); if (c) focar(c, false); });
$("lista").addEventListener("dblclick", (ev) => {
  const c = ev.target.closest(".card");
  if (c && !ev.target.closest("button")) c.querySelector(".copiar").click();
});
// Clique direito abre o menu do card na posição do mouse
$("lista").addEventListener("contextmenu", (ev) => {
  const c = ev.target.closest(".card");
  if (!c || !desktop()) return;
  ev.preventDefault();
  focar(c, false);
  document.dispatchEvent(new Event("scroll")); // fecha menu aberto
  c.querySelector(".mais").click();
  const m = $("menu-card");
  m.style.top = Math.min(ev.clientY, innerHeight - m.offsetHeight - 8) + "px";
  m.style.right = Math.max(8, innerWidth - ev.clientX - m.offsetWidth) + "px";
  m.style.transformOrigin = "top left";
});

document.addEventListener("keydown", (ev) => {
  if (!cofreAberto() || !desktop()) return;
  const k = ev.key.toLowerCase(), mod = ev.ctrlKey || ev.metaKey;
  if (mod && k === "s" && !$("modal").classList.contains("escondida")) { ev.preventDefault(); $("btn-salvar").click(); return; }
  if (modalAberto()) return;
  if (mod && k === "n") { ev.preventDefault(); $("btn-novo").click(); return; }
  if (mod && (k === "f" || k === "k")) { ev.preventDefault(); $("busca").focus(); $("busca").select(); return; }
  if (mod && k === "l") { ev.preventDefault(); $("btn-travar").click(); return; }
  if (mod && k === "b") { ev.preventDefault(); $("btn-mais").click(); itemMenu(/Exportar/)?.click(); return; }
  if (mod || ev.altKey) return;

  const ativo = document.activeElement, digitando = /^(INPUT|TEXTAREA|SELECT)$/.test(ativo?.tagName);
  if (ev.key === "Escape" && ativo === $("busca")) { $("busca").value = ""; $("busca").dispatchEvent(new Event("input")); $("busca").blur(); return; }
  if (!digitando && ev.key === "/") { ev.preventDefault(); $("busca").focus(); return; }
  if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
    if (digitando && ativo !== $("busca")) return;
    const l = cards();
    if (!l.length) return;
    ev.preventDefault();
    let i = l.findIndex((c) => chave(c) === foco);
    i = ev.key === "ArrowDown" ? Math.min(i + 1, l.length - 1) : Math.max(i - 1, 0);
    focar(l[i], true);
    return;
  }
  if (digitando || ativo !== document.body) return;
  const card = cards().find((c) => chave(c) === foco);
  if (!card) return;
  if (ev.key === "Enter") { ev.preventDefault(); card.querySelector(".copiar").click(); }
  else if (k === "e") doMenu(card, /Editar/);
  else if (ev.key === "Delete") doMenu(card, /Apagar/);
  else if (k === "u") $("detalhe").querySelector('[data-a="user"]')?.click();
});

// Contagem até a área de transferência ser limpa (15 s) na barra inferior
let clip = 0;
document.addEventListener("click", (ev) => {
  if (ev.target.closest('.copiar, [data-a="senha"], [data-a="user"]')) clip = 15;
}, true);
setInterval(() => {
  if (clip > 0) clip--;
  $("st-clip").textContent = clip > 0 ? `Área de transferência limpa em ${clip}s` : "";
}, 1000);

// Detalhe vazio ao abrir
atualizarDetalhe();