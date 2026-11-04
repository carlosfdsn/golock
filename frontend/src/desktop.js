import "./desktop.css";
import { CopyToClipboard } from "../wailsjs/go/main/App";

const $ = (id) => document.getElementById(id);
const desktop = () => matchMedia("(min-width: 1000px) and (min-height: 560px)").matches;
const cofreAberto = () => !$("tela-cofre").classList.contains("escondida");
const modalAberto = () =>
  [...document.querySelectorAll(".modal")].some((m) => !m.classList.contains("escondida"));
const cards = () => [...document.querySelectorAll("#lista .card")];
const chave = (c) =>
  c.querySelector("strong").textContent + "\u0000" + c.querySelector(".user").textContent;

// Barra de atalhos + contagem do bloqueio automático
$("tela-cofre").insertAdjacentHTML(
  "beforeend",
  `<footer class="atalhos" aria-hidden="true">
    <span><kbd>Ctrl+N</kbd>nova</span><span><kbd>Ctrl+F</kbd>buscar</span>
    <span><kbd>↑</kbd><kbd>↓</kbd>navegar</span><span><kbd>Enter</kbd>copiar senha</span>
    <span><kbd>U</kbd>usuário</span><span><kbd>E</kbd>editar</span><span><kbd>Del</kbd>apagar</span>
    <span><kbd>Ctrl+L</kbd>travar</span><span class="espaco"></span><span id="st-trava"></span>
  </footer>`,
);

let ultima = Date.now();
["pointerdown", "pointermove", "keydown", "wheel"].forEach((ev) =>
  addEventListener(ev, () => (ultima = Date.now()), { passive: true, capture: true }),
);
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

// Item em foco (sobrevive a re-renderizações da lista)
let foco = null;
function aplicarFoco(rolar) {
  cards().forEach((c) => {
    const on = chave(c) === foco;
    c.classList.toggle("foco", on);
    if (on && rolar) c.scrollIntoView({ block: "nearest" });
  });
}
new MutationObserver(() => aplicarFoco(false)).observe($("lista"), { childList: true });

$("lista").addEventListener("click", (ev) => {
  const c = ev.target.closest(".card");
  if (c) { foco = chave(c); aplicarFoco(false); }
});
// Duplo clique no card copia a senha
$("lista").addEventListener("dblclick", (ev) => {
  const c = ev.target.closest(".card");
  if (c && !ev.target.closest("button")) c.querySelector(".copiar").click();
});

const itemMenu = (re) =>
  [...document.querySelectorAll("#menu-card button")].find((b) => re.test(b.textContent));
const doMenu = (card, re) => { card.querySelector(".mais").click(); itemMenu(re)?.click(); };

// Clique direito abre o menu do card na posição do mouse
$("lista").addEventListener("contextmenu", (ev) => {
  const c = ev.target.closest(".card");
  if (!c || !desktop()) return;
  ev.preventDefault();
  foco = chave(c); aplicarFoco(false);
  document.dispatchEvent(new Event("scroll")); // fecha menu aberto
  c.querySelector(".mais").click();
  const m = $("menu-card");
  m.style.top = Math.min(ev.clientY, innerHeight - m.offsetHeight - 8) + "px";
  m.style.right = Math.max(8, innerWidth - ev.clientX - m.offsetWidth) + "px";
  m.style.transformOrigin = "top left";
});

document.addEventListener("keydown", (ev) => {
  if (!cofreAberto() || !desktop()) return;
  const k = ev.key.toLowerCase();
  const mod = ev.ctrlKey || ev.metaKey;

  if (mod && k === "s" && !$("modal").classList.contains("escondida")) {
    ev.preventDefault(); $("btn-salvar").click(); return;
  }
  if (modalAberto()) return;

  if (mod && k === "n") { ev.preventDefault(); $("btn-novo").click(); return; }
  if (mod && k === "f") { ev.preventDefault(); $("busca").focus(); $("busca").select(); return; }
  if (mod && k === "l") { ev.preventDefault(); $("btn-travar").click(); return; }
  if (mod && k === "b") {
    ev.preventDefault(); $("btn-mais").click(); itemMenu(/Exportar/)?.click(); return;
  }
  if (mod || ev.altKey) return;

  const ativo = document.activeElement;
  const digitando = /^(INPUT|TEXTAREA|SELECT)$/.test(ativo?.tagName);
  if (ev.key === "Escape" && ativo === $("busca")) {
    $("busca").value = ""; $("busca").dispatchEvent(new Event("input")); $("busca").blur(); return;
  }
  if (!digitando && ev.key === "/") { ev.preventDefault(); $("busca").focus(); return; }

  if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
    if (digitando && ativo !== $("busca")) return;
    const lista = cards();
    if (!lista.length) return;
    ev.preventDefault();
    let i = lista.findIndex((c) => chave(c) === foco);
    i = ev.key === "ArrowDown" ? Math.min(i + 1, lista.length - 1) : Math.max(i - 1, 0);
    foco = chave(lista[i]); aplicarFoco(true);
    return;
  }

  if (digitando || ativo !== document.body) return;
  const card = cards().find((c) => chave(c) === foco);
  if (!card) return;
  if (ev.key === "Enter") { ev.preventDefault(); card.querySelector(".copiar").click(); }
  else if (k === "e") doMenu(card, /Editar/);
  else if (ev.key === "Delete") doMenu(card, /Apagar/);
  else if (k === "u") {
    const u = card.querySelector(".user").textContent;
    if (u === "Sem usuário") return aviso("Esta entrada não tem usuário");
    CopyToClipboard(u).then(() => aviso("Usuário copiado · some em 15s"));
  }
});