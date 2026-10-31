import "./style.css";
// Importado (e não escrito como texto): só assim o Vite inclui o arquivo no build
// e troca o caminho pelo nome final. Com "./src/assets/..." em texto, a imagem
// existe no `wails dev`, mas some no app compilado.
import mascoteUrl from "./assets/mascote.png";
import {
  HasVault,
  CreateVault,
  Unlock,
  Lock,
  Categories,
  DetectCategory,
  AddEntry,
  ListEntries,
  UpdateEntry,
  DeleteEntry,
  GeneratePassword,
  CopyToClipboard,
  ExportEntries,
  ImportEntries,
  HasRecoveryKey,
  GenerateRecoveryKey,
  RecoverVault,
} from "../wailsjs/go/main/App";

const $ = (id) => document.getElementById(id);
const MASC = mascoteUrl;

// ---------- Ícones ----------
const P = {
  lock: '<rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  x: '<path d="M18 6 6 18M6 6l12 12"/>',
  sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
  moon: '<path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/>',
  eye: '<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>',
  eyeOff:
    '<path d="M3 3l18 18M10.6 6.1A10 10 0 0 1 12 5c6.5 0 10 7 10 7a17 17 0 0 1-3.2 4M6.6 6.6A17 17 0 0 0 2 12s3.5 7 10 7a9.7 9.7 0 0 0 4-.9"/>',
  copy: '<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1"/>',
  dots: '<circle cx="5" cy="12" r="1.3" fill="currentColor"/><circle cx="12" cy="12" r="1.3" fill="currentColor"/><circle cx="19" cy="12" r="1.3" fill="currentColor"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  edit: '<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/>',
  trash: '<path d="M3 6h18M8 6V4h8v2M6 6l1 14h10l1-14M10 11v5M14 11v5"/>',
  search: '<circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/>',
  wand: '<path d="m12 3 1.9 5.1L19 10l-5.1 1.9L12 17l-1.9-5.1L5 10l5.1-1.9z"/><path d="M19 17v4M17 19h4"/>',
  check: '<path d="m5 12.5 4.5 4.5L19 7"/>',
  shield:
    '<path d="M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6z"/><path d="m9 12 2 2 4-4"/>',
  alert: '<path d="M12 4 2.5 20h19z"/><path d="M12 10v4M12 17h.01"/>',
  arrow: '<path d="M5 12h14M13 6l6 6-6 6"/>',
  user: '<circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/>',
  tag: '<path d="M3 12V4a1 1 0 0 1 1-1h8l9 9-9 9z"/><circle cx="8" cy="8" r="1.2"/>',
  layers: '<path d="m12 3 9 5-9 5-9-5z"/><path d="m3 13 9 5 9-5"/>',
  key: '<circle cx="8" cy="15" r="4"/><path d="m11 12 9-9M16 7l3 3"/>',
  email:
    '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3 7 9 6 9-6"/>',
  banco: '<path d="m3 10 9-6 9 6M5 10v8M9 10v8M15 10v8M19 10v8M3 21h18"/>',
  social:
    '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20a6.5 6.5 0 0 1 13 0M16 4.5a3.5 3.5 0 0 1 0 7M18 14a6.5 6.5 0 0 1 3.5 6"/>',
  streaming: '<circle cx="12" cy="12" r="9"/><path d="m10 8.5 5 3.5-5 3.5z"/>',
  jogos:
    '<rect x="2" y="7" width="20" height="11" rx="5"/><path d="M7 10v5M4.5 12.5h5M16 11.5h.01M18 14h.01"/>',
  trabalho:
    '<rect x="3" y="7" width="18" height="13" rx="2"/><path d="M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2M3 13h18"/>',
  compras: '<path d="M6 7h12l1 13H5zM9 7a3 3 0 0 1 6 0"/>',
  outros: '<circle cx="8" cy="15" r="4"/><path d="m11 12 9-9M16 7l3 3"/>',
  archive:
    '<rect x="3" y="4" width="18" height="5" rx="1"/><path d="M5 9v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V9M10 13h4"/>',
  download: '<path d="M12 3v12M7 10l5 5 5-5M4 20h16"/>',
  upload: '<path d="M12 16V4M7 9l5-5 5 5M4 20h16"/>',
  file: '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/>',
  table:
    '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 10h18M9 4v16"/>',
  lines: '<path d="M4 6h16M4 12h16M4 18h10"/>',
};
const svg = (n) =>
  `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${P[n] || P.outros}</svg>`;

document
  .querySelectorAll("[data-i]")
  .forEach((el) => (el.innerHTML = svg(el.dataset.i)));

// ---------- Mascote ----------
function ajustarMascotes(raiz = document) {
  raiz.querySelectorAll("img.mascote").forEach((img) => {
    const trocar = () => {
      const s = document.createElement("span");
      s.className = "mascote-fallback";
      s.innerHTML = svg("lock");
      img.replaceWith(s);
    };
    img.addEventListener("error", trocar, { once: true });
    if (img.complete && img.naturalWidth === 0) trocar();
  });
}
ajustarMascotes();

function criarMascote(classe) {
  const w = document.createElement("div");
  w.className = "mascote-wrap " + classe;
  w.innerHTML = `<div class="aura"></div><img class="mascote" src="${MASC}" alt="" />`;
  ajustarMascotes(w);
  return w;
}

function reagir(tipo) {
  document.querySelectorAll(".mascote-wrap").forEach((w) => {
    w.classList.remove("feliz", "erro");
    void w.offsetWidth;
    w.classList.add(tipo);
    setTimeout(() => w.classList.remove(tipo), 800);
  });
}

// Faíscas douradas e roxas saindo de um ponto
function fagulhas(x, y, n = 16) {
  for (let i = 0; i < n; i++) {
    const f = document.createElement("span");
    const ang = Math.random() * Math.PI * 2;
    const dist = 40 + Math.random() * 70;
    f.className = "fagulha";
    f.style.left = x + "px";
    f.style.top = y + "px";
    f.style.background =
      Math.random() > 0.5 ? "var(--ouro-claro)" : "var(--roxo)";
    f.style.setProperty("--dx", Math.cos(ang) * dist + "px");
    f.style.setProperty("--dy", Math.sin(ang) * dist + "px");
    document.body.appendChild(f);
    setTimeout(() => f.remove(), 850);
  }
}
const faiscarDe = (el, n) => {
  const r = el.getBoundingClientRect();
  fagulhas(r.left + r.width / 2, r.top + r.height / 2, n);
};

// Clicar no mascote faz ele pular
document.addEventListener(
  "click",
  (e) => e.target.closest(".mascote-wrap") && reagir("feliz"),
);

// ---------- Tema ----------
function setTema(t) {
  document.documentElement.dataset.tema = t;
  try {
    localStorage.setItem("tema", t);
  } catch {}
  document
    .querySelectorAll(".btn-tema")
    .forEach((b) => (b.innerHTML = svg(t === "dark" ? "sun" : "moon")));
}
let salvo = null;
try {
  salvo = localStorage.getItem("tema");
} catch {}
setTema(
  salvo ||
    (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"),
);
document
  .querySelectorAll(".btn-tema")
  .forEach(
    (b) =>
      (b.onclick = () =>
        setTema(
          document.documentElement.dataset.tema === "dark" ? "light" : "dark",
        )),
  );

// ---------- Utilidades ----------
let toastTimer;
function toast(msg, icone = "check") {
  $("toast-ic").innerHTML = svg(icone);
  $("toast-txt").textContent = msg;
  $("toast").classList.add("mostra");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => $("toast").classList.remove("mostra"), 2000);
}

function mostrar(id) {
  $("tela-login").classList.add("escondida");
  $("tela-cofre").classList.add("escondida");
  $(id).classList.remove("escondida");
}

// Força da senha: 1 (fraca) a 4 (forte), 0 = vazio
const ROTULOS = ["", "Fraca", "Razoável", "Boa", "Forte"];
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
function mostrarForca(el, valor) {
  const n = forca(valor);
  el.dataset.n = n;
  el.querySelector("small").textContent = ROTULOS[n];
}

function contar(el, alvo) {
  const ini = Number(el.textContent) || 0;
  if (ini === alvo) {
    el.textContent = alvo;
    return;
  }
  const t0 = performance.now();
  const passo = (t) => {
    const p = Math.min(1, (t - t0) / 500);
    el.textContent = Math.round(ini + (alvo - ini) * (1 - Math.pow(1 - p, 3)));
    if (p < 1) requestAnimationFrame(passo);
  };
  requestAnimationFrame(passo);
}

const saudacao = () => {
  const h = new Date().getHours();
  return h < 12 ? "Bom dia" : h < 18 ? "Boa tarde" : "Boa noite";
};

// Botões de mostrar/ocultar senha
document.querySelectorAll(".olho").forEach(
  (b) =>
    (b.onclick = () => {
      const input = $(b.dataset.alvo);
      const oculto = input.type === "password";
      input.type = oculto ? "text" : "password";
      b.innerHTML = svg(oculto ? "eyeOff" : "eye");
    }),
);

// ---------- Login ----------
let modoCriar = true;

// ---------- Exportar e importar ----------
let formato = "pdf";

function setFormato(f) {
  formato = f;
  document
    .querySelectorAll("#b-formatos .tile")
    .forEach((t) => t.classList.toggle("sel", t.dataset.f === f));
}
document
  .querySelectorAll("#b-formatos .tile")
  .forEach((t) => (t.onclick = () => setFormato(t.dataset.f)));

function abrirBackup() {
  $("b-senha").value = "";
  $("b-senha").type = "password";
  document.querySelector('[data-alvo="b-senha"]').innerHTML = svg("eye");
  $("b-erro").textContent = "";
  $("b-res").textContent = "";
  setFormato("pdf");
  $("backup").classList.remove("escondida");
  setTimeout(() => $("b-senha").focus(), 60);
}
const fecharBackup = () => $("backup").classList.add("escondida");

$("b-fechar").onclick = fecharBackup;
$("backup").addEventListener(
  "click",
  (e) => e.target === $("backup") && fecharBackup(),
);
$("b-senha").addEventListener(
  "keydown",
  (e) => e.key === "Enter" && $("b-exportar").click(),
);

$("b-exportar").onclick = async () => {
  $("b-erro").textContent = "";
  $("b-res").textContent = "";
  const senha = $("b-senha").value;
  if (!senha) {
    $("b-erro").textContent = "Digite a senha mestra para confirmar.";
    return;
  }
  $("b-exportar").disabled = true;
  try {
    const caminho = await ExportEntries(formato, senha);
    $("b-senha").value = "";
    if (!caminho) return; // cancelou a janela de salvar
    reagir("feliz");
    toast("Exportado com sucesso");
    $("b-res").textContent = "Salvo em: " + caminho;
  } catch (err) {
    $("b-erro").textContent = String(err);
  } finally {
    $("b-exportar").disabled = false;
  }
};

$("b-importar").onclick = async () => {
  $("b-erro").textContent = "";
  $("b-res").textContent = "";
  try {
    const r = await ImportEntries();
    if (!r || !r.format) return; // cancelou
    await carregar();
    reagir("feliz");
    toast(
      `${r.added} ${r.added === 1 ? "senha importada" : "senhas importadas"}`,
    );
    $("b-res").textContent =
      `${r.format}: ${r.added} importadas, ${r.skipped} já existiam, ${r.invalid} ignoradas (sem nome, sem senha ou grandes demais).`;
  } catch (err) {
    $("b-erro").textContent = String(err);
  }
};

async function iniciar() {
  modoCriar = !(await HasVault());
  ["senha", "senha2"].forEach((id) => {
    $(id).value = "";
    $(id).type = "password";
  });
  document
    .querySelectorAll("#tela-login .olho")
    .forEach((b) => (b.innerHTML = svg("eye")));
  $("erro").textContent = "";
  mostrarForca($("forca-login"), "");

  $("subtitulo").textContent = modoCriar
    ? "Crie uma senha mestra. Ela protege todas as outras."
    : "Bem-vindo de volta. Digite sua senha mestra.";
  $("btn-entrar-txt").textContent = modoCriar ? "Criar cofre" : "Desbloquear";
  $("campo-senha2").classList.toggle("escondida", !modoCriar);
  $("forca-login").classList.toggle("escondida", !modoCriar);
  $("btn-esqueci").classList.toggle("escondida", modoCriar);
  mostrar("tela-login");
}

function falhar(msg, input) {
  $("erro").textContent = msg;
  reagir("erro");
  input.classList.remove("tremer");
  void input.offsetWidth;
  input.classList.add("tremer");
}

let entrando = false;
async function entrar() {
  // Enter + clique (ou clique duplo) disparavam duas chamadas: no "Criar cofre"
  // a segunda falhava com "já existe um cofre" por cima da primeira.
  if (entrando) return;
  entrando = true;
  $("btn-entrar").disabled = true;
  $("erro").textContent = "";
  const senha = $("senha").value;
  try {
    let chave = "";
    if (modoCriar) {
      if (senha !== $("senha2").value) {
        falhar("As senhas não são iguais.", $("senha2"));
        return;
      }
      chave = await CreateVault(senha);
    } else {
      await Unlock(senha);
    }
    reagir("feliz");
    fagulhas(innerWidth / 2, innerHeight / 3, 26);
    await new Promise((r) => setTimeout(r, 650));
    if (chave) await mostrarChave(chave);
    await abrirCofre();
  } catch (err) {
    falhar(String(err), $("senha"));
  } finally {
    entrando = false;
    $("btn-entrar").disabled = false;
  }
}

$("btn-entrar").onclick = entrar;
["senha", "senha2"].forEach((id) => {
  $(id).addEventListener("keydown", (e) => e.key === "Enter" && entrar());
  $(id).addEventListener("focus", () =>
    $("mascote-login").classList.add("curioso"),
  );
  $(id).addEventListener("blur", () =>
    $("mascote-login").classList.remove("curioso"),
  );
});
$("senha").addEventListener("input", () =>
  mostrarForca($("forca-login"), $("senha").value),
);

let travando = false;
async function travar(auto = false) {
  if (travando) return;
  travando = true;
  try {
    fecharMenu();
    fecharAuto();
    fecharBackup();
    fecharGerar();
    fecharModal();
    fecharEsqueci();
    $("confirma").classList.add("escondida");
    await Lock();
    entries = [];
    revelados.clear();
    $("busca").value = "";
    await iniciar();
    if (auto) toast("Bloqueado por inatividade", "lock");
  } finally {
    travando = false;
  }
}
$("btn-travar").onclick = () => travar();

// ---------- Bloqueio por inatividade ----------
const TEMPOS = [1, 5, 15, 30]; // minutos
let minutosAuto = 5;
try {
  const v = Number(localStorage.getItem("autolock"));
  if (TEMPOS.includes(v)) minutosAuto = v;
} catch {}
let ultimaAtividade = Date.now();
const marcarAtividade = () => (ultimaAtividade = Date.now());
["pointerdown", "pointermove", "keydown", "wheel", "touchstart"].forEach((ev) =>
  addEventListener(ev, marcarAtividade, { passive: true, capture: true }),
);

function verificarInatividade() {
  if ($("tela-cofre").classList.contains("escondida")) return;
  // com a chave de recuperação na tela a pessoa precisa de tempo para anotá-la
  if (!$("recup").classList.contains("escondida")) return marcarAtividade();
  if (Date.now() - ultimaAtividade >= minutosAuto * 60000) travar(true);
}
setInterval(verificarInatividade, 1000);
document.addEventListener("visibilitychange", verificarInatividade);

function marcarTempoAtual() {
  document
    .querySelectorAll("#a-tempos .tile")
    .forEach((t) =>
      t.classList.toggle("sel", Number(t.dataset.m) === minutosAuto),
    );
}
function abrirAuto() {
  marcarTempoAtual();
  $("auto").classList.remove("escondida");
}
const fecharAuto = () => $("auto").classList.add("escondida");
document.querySelectorAll("#a-tempos .tile").forEach(
  (t) =>
    (t.onclick = () => {
      minutosAuto = Number(t.dataset.m);
      try {
        localStorage.setItem("autolock", String(minutosAuto));
      } catch {}
      marcarAtividade();
      fecharAuto();
      toast(`Trava após ${minutosAuto} min sem uso`, "lock");
    }),
);
$("a-fechar").onclick = fecharAuto;
$("auto").addEventListener(
  "click",
  (e) => e.target === $("auto") && fecharAuto(),
);

// ---------- Menu do cabeçalho (⋯) ----------
$("btn-mais").onclick = (ev) => {
  const escuro = document.documentElement.dataset.tema === "dark";
  abrirMenu(ev.currentTarget, [
    {
      icone: escuro ? "sun" : "moon",
      texto: escuro ? "Tema claro" : "Tema escuro",
      fn: () => setTema(escuro ? "light" : "dark"),
    },
    { icone: "key", texto: "Chave de recuperação", fn: abrirGerar },
    { icone: "archive", texto: "Exportar e importar", fn: abrirBackup },
    "-",
    { icone: "clock", texto: "Bloqueio automático", fn: abrirAuto },
  ]);
};

// ---------- Chave de recuperação ----------

// Mostra a chave (uma única vez) e só continua quando a pessoa confirmar que guardou.
function mostrarChave(chave) {
  return new Promise((resolve) => {
    const g = chave.split("-");
    $("recup-chave").textContent =
      g.slice(0, 4).join("-") + "\n" + g.slice(4).join("-");
    $("recup-ok").checked = false;
    $("recup-continuar").disabled = true;
    $("recup").classList.remove("escondida");

    $("recup-copiar").onclick = async () => {
      await CopyToClipboard(chave);
      toast("Copiada · some em 15s");
    };
    $("recup-ok").onchange = () => {
      $("recup-continuar").disabled = !$("recup-ok").checked;
    };
    $("recup-continuar").onclick = () => {
      $("recup").classList.add("escondida");
      $("recup-chave").textContent = "";
      resolve();
    };
  });
}

// ---------- Esqueci a senha mestra ----------
function limparEsqueci() {
  ["e-chave", "e-senha", "e-senha2"].forEach((id) => {
    $(id).value = "";
  });
  ["e-senha", "e-senha2"].forEach((id) => {
    $(id).type = "password";
    document.querySelector(`[data-alvo="${id}"]`).innerHTML = svg("eye");
  });
  $("e-erro").textContent = "";
  mostrarForca($("forca-e"), "");
}

async function abrirEsqueci() {
  limparEsqueci();
  const tem = await HasRecoveryKey();
  $("e-sem").classList.toggle("escondida", tem);
  $("e-form").classList.toggle("escondida", !tem);
  $("esqueci").classList.remove("escondida");
  if (tem) setTimeout(() => $("e-chave").focus(), 60);
}
const fecharEsqueci = () => {
  limparEsqueci();
  $("esqueci").classList.add("escondida");
};

$("btn-esqueci").onclick = abrirEsqueci;
$("e-fechar").onclick = fecharEsqueci;
$("esqueci").addEventListener(
  "click",
  (e) => e.target === $("esqueci") && fecharEsqueci(),
);
$("e-senha").addEventListener("input", () =>
  mostrarForca($("forca-e"), $("e-senha").value),
);
["e-chave", "e-senha", "e-senha2"].forEach((id) =>
  $(id).addEventListener(
    "keydown",
    (e) => e.key === "Enter" && $("e-ok").click(),
  ),
);

$("e-ok").onclick = async () => {
  $("e-erro").textContent = "";
  if (!$("e-chave").value.trim()) {
    $("e-erro").textContent = "Digite a chave de recuperação.";
    return;
  }
  if ($("e-senha").value !== $("e-senha2").value) {
    $("e-erro").textContent = "As senhas não são iguais.";
    reagir("erro");
    return;
  }
  $("e-ok").disabled = true;
  $("e-ok-txt").textContent = "Redefinindo…";
  try {
    const nova = await RecoverVault($("e-chave").value, $("e-senha").value);
    fecharEsqueci();
    reagir("feliz");
    fagulhas(innerWidth / 2, innerHeight / 3, 26);
    await mostrarChave(nova);
    await abrirCofre();
    toast("Senha redefinida");
  } catch (err) {
    $("e-erro").textContent = String(err);
    reagir("erro");
  } finally {
    $("e-ok").disabled = false;
    $("e-ok-txt").textContent = "Redefinir senha";
  }
};

// ---------- Gerar chave de recuperação (dentro do cofre) ----------
function limparGerar() {
  $("g-senha").value = "";
  $("g-senha").type = "password";
  document.querySelector('[data-alvo="g-senha"]').innerHTML = svg("eye");
  $("g-erro").textContent = "";
}

async function abrirGerar() {
  limparGerar();
  const tem = await HasRecoveryKey();
  $("g-status").textContent = tem
    ? "Seu cofre já tem uma chave de recuperação. Se perdeu a anterior ou quer trocar, gere outra."
    : "Seu cofre ainda não tem chave de recuperação. Gere uma para conseguir voltar aos seus dados se esquecer a senha mestra.";
  $("g-ok-txt").textContent = tem ? "Gerar nova chave" : "Gerar chave";
  $("gerar").classList.remove("escondida");
  setTimeout(() => $("g-senha").focus(), 60);
}
const fecharGerar = () => {
  limparGerar();
  $("gerar").classList.add("escondida");
};

$("g-fechar").onclick = fecharGerar;
$("gerar").addEventListener(
  "click",
  (e) => e.target === $("gerar") && fecharGerar(),
);
$("g-senha").addEventListener(
  "keydown",
  (e) => e.key === "Enter" && $("g-ok").click(),
);

$("g-ok").onclick = async () => {
  $("g-erro").textContent = "";
  if (!$("g-senha").value) {
    $("g-erro").textContent = "Digite a senha mestra para confirmar.";
    return;
  }
  $("g-ok").disabled = true;
  try {
    const chave = await GenerateRecoveryKey($("g-senha").value);
    fecharGerar();
    await mostrarChave(chave);
    toast("Chave de recuperação atualizada");
  } catch (err) {
    $("g-erro").textContent = String(err);
  } finally {
    $("g-ok").disabled = false;
  }
};

// ---------- Cofre ----------
let cats = [];
let entries = [];
let filtro = "todas";
const revelados = new Set();

async function abrirCofre() {
  marcarAtividade();
  cats = await Categories();
  filtro = "todas";
  $("busca").value = "";
  $("saudacao").textContent = saudacao();
  await carregar();
  mostrar("tela-cofre");
  // cofre criado antes da chave de recuperação existir: oferece gerar uma
  if (!(await HasRecoveryKey())) abrirGerar();
}

async function carregar() {
  entries = (await ListEntries()) || [];
  render(true);
}

function atualizarResumo() {
  const fracas = entries.filter((e) => forca(e.password) <= 2).length;
  contar($("st-total"), entries.length);
  contar($("st-cats"), new Set(entries.map((e) => e.category)).size);
  contar($("st-fracas"), fracas);
  document.querySelector(".stat.aviso").classList.toggle("tem", fracas > 0);
}

function render(animar = false) {
  fecharMenu();
  const termo = $("busca").value.trim().toLowerCase();
  atualizarResumo();

  // chips
  const chips = $("chips");
  const rolagem = chips.scrollLeft; // mantém a posição ao trocar de filtro
  chips.innerHTML = "";
  const addChip = (key, label, icone, n) => {
    const b = document.createElement("button");
    b.className = "chip" + (filtro === key ? " ativo" : "");
    b.innerHTML = svg(icone) + "<span></span>";
    b.querySelector("span").textContent = `${label} · ${n}`;
    b.onclick = () => {
      filtro = key;
      render(true);
    };
    chips.appendChild(b);
  };
  addChip("todas", "Todas", "layers", entries.length);
  cats.forEach((c) => {
    const n = entries.filter((e) => e.category === c.key).length;
    if (n) addChip(c.key, c.label, c.key, n);
  });
  chips.scrollLeft = rolagem;
  if (filtro !== "todas" && !entries.some((e) => e.category === filtro))
    filtro = "todas";

  // lista agrupada por categoria
  const lista = $("lista");
  lista.innerHTML = "";
  let idx = 0;

  cats.forEach((c) => {
    if (filtro !== "todas" && filtro !== c.key) return;
    const itens = entries.filter(
      (e) =>
        e.category === c.key &&
        (!termo ||
          e.name.toLowerCase().includes(termo) ||
          e.username.toLowerCase().includes(termo)),
    );
    if (!itens.length) return;

    const grupo = document.createElement("section");
    grupo.className = "grupo";
    grupo.innerHTML = `<div class="grupo-titulo">${svg(c.key)}<span>${c.label}</span></div>`;
    itens.forEach((e) => grupo.appendChild(criarCard(e, idx++, animar)));
    lista.appendChild(grupo);
  });

  if (!lista.children.length) {
    const vazio = document.createElement("div");
    vazio.className = "vazio";
    vazio.appendChild(criarMascote("medio"));
    const t = document.createElement("strong");
    const p = document.createElement("p");
    if (entries.length) {
      t.textContent = "Nada encontrado";
      p.textContent = "Tente outro nome ou categoria.";
    } else {
      t.textContent = "Seu cofre está vazio";
      p.textContent = "Toque no + e guarde sua primeira senha.";
    }
    vazio.append(t, p);
    lista.appendChild(vazio);
  }
}

// ---------- Menu de ações do card (⋯) ----------
const menu = document.createElement("div");
menu.id = "menu-card";
menu.className = "menu-card escondida";
menu.setAttribute("role", "menu");
document.body.appendChild(menu);
let menuBtn = null; // botão ⋯ que abriu o menu

function fecharMenu() {
  menu.classList.add("escondida");
  menu.innerHTML = "";
  if (menuBtn) menuBtn.setAttribute("aria-expanded", "false");
  menuBtn = null;
}

// itens: [{ icone, texto, perigo, fn }] — "-" desenha um separador
function abrirMenu(btn, itens) {
  if (menuBtn === btn) return fecharMenu(); // clicar de novo fecha
  fecharMenu();
  itens.forEach((it) => {
    if (it === "-") {
      menu.appendChild(document.createElement("hr"));
      return;
    }
    const b = document.createElement("button");
    b.type = "button";
    b.setAttribute("role", "menuitem");
    if (it.perigo) b.className = "perigo";
    b.innerHTML = svg(it.icone) + "<span></span>";
    b.querySelector("span").textContent = it.texto;
    b.onclick = () => {
      fecharMenu();
      it.fn();
    };
    menu.appendChild(b);
  });
  menuBtn = btn;
  btn.setAttribute("aria-expanded", "true");

  // posiciona perto do botão; vira para cima se não couber embaixo
  menu.classList.remove("escondida");
  const r = btn.getBoundingClientRect();
  const h = menu.offsetHeight;
  const abaixo = r.bottom + 6 + h <= innerHeight - 8;
  menu.style.top = (abaixo ? r.bottom + 6 : Math.max(8, r.top - h - 6)) + "px";
  menu.style.right = Math.max(8, innerWidth - r.right) + "px";
  menu.style.transformOrigin = abaixo ? "top right" : "bottom right";
}

document.addEventListener("pointerdown", (ev) => {
  if (menuBtn && !menu.contains(ev.target) && !ev.target.closest(".mais"))
    fecharMenu();
});
addEventListener("resize", fecharMenu);
document.addEventListener("scroll", fecharMenu, true);

function criarCard(e, i, animar) {
  const card = document.createElement("div");
  card.className = "card" + (animar ? " entra" : "");
  card.style.setProperty("--i", i);
  card.addEventListener("pointermove", (ev) => {
    const r = card.getBoundingClientRect();
    card.style.setProperty("--mx", ev.clientX - r.left + "px");
    card.style.setProperty("--my", ev.clientY - r.top + "px");
  });

  const visivel = revelados.has(e.id);
  const fraca = forca(e.password) <= 2;

  // Só o essencial fica à vista: copiar e o menu ⋯ (ver, editar, apagar)
  card.innerHTML = `
    <div class="avatar">${svg(e.category)}</div>
    <div class="info">
      <strong></strong>
      <small class="user"></small>
      <small class="senha"></small>
    </div>
    <div class="mini">
      ${fraca ? `<span class="alerta" title="Senha fraca">${svg("alert")}</span>` : ""}
      <button class="icon-btn copiar" aria-label="Copiar senha" title="Copiar senha">${svg("copy")}</button>
      <button class="icon-btn mais" aria-label="Mais opções" title="Mais opções" aria-haspopup="menu" aria-expanded="false">${svg("dots")}</button>
    </div>`;

  // textContent: seguro contra HTML digitado pelo usuário
  card.querySelector("strong").textContent = e.name;
  card.querySelector(".user").textContent = e.username || "Sem usuário";
  card.querySelector(".senha").textContent = visivel
    ? e.password
    : "••••••••••";

  const apagar = async () => {
    if (!(await confirmar(e.name))) return;
    card.classList.remove("entra");
    card.classList.add("saindo");
    await new Promise((r) => setTimeout(r, 350));
    try {
      await DeleteEntry(e.id);
      revelados.delete(e.id);
      await carregar();
      toast("Senha apagada", "trash");
    } catch (err) {
      // não gravou no disco: a senha continua no cofre e a tela volta a mostrá-la
      await carregar();
      toast(String(err), "alert");
    }
  };

  card.querySelector(".copiar").onclick = async (ev) => {
    const btn = ev.currentTarget;
    await CopyToClipboard(e.password);
    reagir("feliz");
    faiscarDe(btn, 10);
    btn.innerHTML = svg("check");
    btn.classList.add("ok");
    setTimeout(() => {
      btn.innerHTML = svg("copy");
      btn.classList.remove("ok");
    }, 1500);
    toast("Copiada · some em 15s");
  };

  card.querySelector(".mais").onclick = (ev) =>
    abrirMenu(ev.currentTarget, [
      {
        icone: visivel ? "eyeOff" : "eye",
        texto: visivel ? "Ocultar senha" : "Mostrar senha",
        fn: () => {
          visivel ? revelados.delete(e.id) : revelados.add(e.id);
          render(false);
        },
      },
      { icone: "edit", texto: "Editar", fn: () => abrirModal(e) },
      "-",
      { icone: "trash", texto: "Apagar", perigo: true, fn: apagar },
    ]);

  return card;
}

$("busca").addEventListener("input", () => render(false));

// Roda do mouse sobre as categorias rola para os lados
$("chips").addEventListener(
  "wheel",
  (ev) => {
    const el = ev.currentTarget;
    if (el.scrollWidth <= el.clientWidth) return; // tudo cabe: nada a rolar
    const d = Math.abs(ev.deltaX) > Math.abs(ev.deltaY) ? ev.deltaX : ev.deltaY;
    ev.preventDefault();
    el.scrollLeft += d;
  },
  { passive: false },
);

// ---------- Confirmar ----------
function confirmar(nome) {
  return new Promise((resolve) => {
    $("conf-nome").textContent = nome;
    $("confirma").classList.remove("escondida");
    const fim = (v) => {
      $("confirma").classList.add("escondida");
      resolve(v);
    };
    $("conf-sim").onclick = () => fim(true);
    $("conf-nao").onclick = () => fim(false);
  });
}

// ---------- Modal de adicionar ----------
let catManual = ""; // '' = automático
let detectada = "";
let editandoId = null; // null = nova senha; número = editando essa senha

function renderTiles() {
  const box = $("f-cats");
  box.innerHTML = "";
  cats.forEach((c) => {
    const b = document.createElement("button");
    b.type = "button";
    b.className =
      "tile" +
      (catManual === c.key ? " sel" : "") +
      (!catManual && detectada === c.key ? " auto" : "");
    b.innerHTML = svg(c.key) + "<span></span>";
    b.querySelector("span").textContent = c.label;
    b.onclick = () => {
      catManual = catManual === c.key ? "" : c.key;
      renderTiles();
      atualizarDica();
    };
    box.appendChild(b);
  });
}

function atualizarDica() {
  const dica = $("dica");
  const c = cats.find((x) => x.key === (catManual || detectada));
  if (!c) {
    dica.innerHTML = "";
    return;
  }
  dica.innerHTML = svg(catManual ? "check" : "wand") + "<span></span>";
  dica.lastChild.textContent = catManual
    ? `Categoria: ${c.label}`
    : `Detectado automaticamente: ${c.label}`;
}

// abrirModal() = nova senha · abrirModal(entrada) = editar essa senha
async function abrirModal(entrada) {
  editandoId = entrada ? entrada.id : null;
  $("f-nome").value = entrada ? entrada.name : "";
  $("f-user").value = entrada ? entrada.username : "";
  $("f-pass").value = entrada ? entrada.password : "";
  $("f-pass").type = "password";
  $("btn-ver").innerHTML = svg("eye");
  mostrarForca($("forca-modal"), $("f-pass").value);
  catManual = entrada ? entrada.category : "";
  detectada = "";
  $("f-erro").textContent = "";
  $("modal-titulo").textContent = entrada ? "Editar senha" : "Nova senha";
  $("btn-salvar-txt").textContent = entrada
    ? "Salvar alterações"
    : "Salvar no cofre";
  renderTiles();
  atualizarDica();
  $("modal").classList.remove("escondida");
  setTimeout(() => $("f-nome").focus(), 60);
  if (entrada) {
    // sugestão automática pelo nome atual (aparece se a pessoa desmarcar a categoria)
    const key = await DetectCategory(entrada.name);
    if (editandoId === entrada.id) {
      detectada = key;
      renderTiles();
    }
  }
}
function fecharModal() {
  $("modal").classList.add("escondida");
  editandoId = null;
  $("f-pass").value = ""; // não deixa a senha na tela/DOM depois de fechar
}

$("btn-novo").onclick = () => abrirModal();
$("btn-fechar").onclick = fecharModal;
$("modal").addEventListener(
  "click",
  (e) => e.target === $("modal") && fecharModal(),
);

$("f-nome").addEventListener("input", async () => {
  const nome = $("f-nome").value.trim();
  if (!nome) {
    detectada = "";
  } else {
    const key = await DetectCategory(nome);
    if ($("f-nome").value.trim() !== nome) return; // digitou mais, descarta
    detectada = key;
  }
  renderTiles();
  atualizarDica();
});

$("f-pass").addEventListener("input", () =>
  mostrarForca($("forca-modal"), $("f-pass").value),
);

$("btn-gerar").onclick = async () => {
  const s = await GeneratePassword(20);
  $("f-pass").value = s;
  $("f-pass").type = "text";
  $("btn-ver").innerHTML = svg("eyeOff");
  mostrarForca($("forca-modal"), s);
};

async function salvar() {
  $("f-erro").textContent = "";
  const editando = editandoId !== null;
  try {
    if (editando) {
      await UpdateEntry(
        editandoId,
        $("f-nome").value,
        $("f-user").value,
        $("f-pass").value,
        catManual,
      );
    } else {
      await AddEntry(
        $("f-nome").value,
        $("f-user").value,
        $("f-pass").value,
        catManual,
      );
    }
    faiscarDe($("btn-salvar"), 18);
    fecharModal();
    await carregar();
    toast(editando ? "Alterações salvas" : "Salva no cofre");
  } catch (err) {
    $("f-erro").textContent = String(err);
  }
}
$("btn-salvar").onclick = salvar;

document.addEventListener("keydown", (e) => {
  if (e.key !== "Escape") return;
  if (menuBtn) {
    fecharMenu();
    return;
  }
  if (!$("recup").classList.contains("escondida")) return; // só fecha ao confirmar que guardou a chave
  if (!$("esqueci").classList.contains("escondida")) fecharEsqueci();
  else if (!$("gerar").classList.contains("escondida")) fecharGerar();
  else if (!$("confirma").classList.contains("escondida"))
    $("conf-nao").click();
  else if (!$("auto").classList.contains("escondida")) fecharAuto();
  else if (!$("backup").classList.contains("escondida")) fecharBackup();
  else fecharModal();
});

iniciar();