// Wspólne parametry i generatory dla skryptów k6 (wszystko deterministyczne: bez Math.random).
import { SharedArray } from "k6/data";
import grpc from "k6/net/grpc";
import { Gauge } from "k6/metrics";

// Rozmiar ładunku napastnika (seria C i D): bajty treści żądania (HTTP) albo zakodowanej wiadomości (gRPC) oraz liczba
// pozycji zamówienia. Gauge -> w podsumowaniu k6 (summary-export) ostatnia/min/max wartość.
export const attackPayloadBytes = new Gauge("attack_payload_bytes");
export const attackPayloadItems = new Gauge("attack_payload_items");

export const IFACE = __ENV.IFACE || "rest";   // rest | graphql | grpc
export const OP = __ENV.OP || "O1";           // O1 | O2 | O3 | O4
export const RATE = parseInt(__ENV.RATE || "100");
export const HOST = __ENV.HOST || "localhost:8443"; // host:port bez schematu
export const TLS = (__ENV.TLS || "false") === "true";
export const AUTH = (__ENV.AUTH || "false") === "true";
export const WARMUP = __ENV.WARMUP || "120s";    // rozgrzewka (odrzucana w analizie)
export const DURATION = __ENV.DURATION || "600s"; // właściwy pomiar
export const COOLDOWN = __ENV.COOLDOWN || "120s"; // wystygnięcie po pomiarze

// "90s" | "10m" | "1h" -> sekundy
export function secs(v) {
  const m = String(v).match(/^(\d+)(s|m|h)?$/);
  return parseInt(m[1]) * ({ s: 1, m: 60, h: 3600 }[m[2] || "s"]);
}
export const N_USERS = 10000;

export const BASE = `${TLS ? "https" : "http"}://${HOST}`;

const tokens = new SharedArray("tokens", () => {
  if (!AUTH) return [{ legit: [], attacker: "" }];
  return [JSON.parse(open(__ENV.TOKENS || "../tokens.json"))];
});

export function legitToken() {
  if (!AUTH) return "";
  const l = tokens[0].legit;
  return l[(__VU * 7919 + __ITER) % l.length];
}
export function attackerToken() { return AUTH ? tokens[0].attacker : ""; }

export function headers(token) {
  const h = { "Content-Type": "application/json" };
  if (token) h["Authorization"] = `Bearer ${token}`;
  return h;
}

// deterministyczny identyfikator z całego zakresu (nie zawsze ten sam rekord)
export function pickId() {
  return 1 + ((__VU * 104729 + __ITER * 7919) % N_USERS);
}

export const GQL = {
  O1: `query($id: ID!){ user(id:$id){ id name email } }`,
  O2: `query{ orders(status:PAID, limit:20, offset:0){ id status createdAt } }`,
  O3: `query($id: ID!){ user(id:$id){ id name orders(limit:10){ id items{ quantity product{ id name price } } } } }`,
  O4: `mutation($in: NewOrder!){ createOrder(input:$in){ id } }`,
};

export function newOrderBody(userId, nItems) {
  const items = [];
  for (let i = 0; i < nItems; i++) items.push({ productId: 1 + ((userId + i * 31) % 5000), quantity: 1 + (i % 5) });
  return { userId, items };
}

// ---- generatory ładunków (seria C, D) ----

// zagnieżdżony JSON o zadanej głębokości (zamówienie + dodatkowe pole x)
export function nestedJson(userId, depth) {
  let x = 1;
  for (let i = 0; i < depth; i++) x = { a: x };
  return JSON.stringify({ ...newOrderBody(userId, 1), x });
}

// zapytanie GraphQL o zadanej głębokości: user.orders.items.product.category.parent.parent...
export function deepQuery(depth) {
  const chain = ["orders(limit:5)", "items", "product", "category"];
  let inner = "id";
  const path = [];
  for (let i = 0; i < depth - 1; i++) path.push(i < chain.length ? chain[i] : "parent");
  for (let i = path.length - 1; i >= 0; i--) inner = `${path[i]}{ ${inner} }`;
  return `query($id: ID!){ user(id:$id){ ${inner} } }`;
}

export function aliasQuery(n) {
  const parts = [];
  for (let i = 0; i < n; i++) parts.push(`a${i}: user(id:$id){ orders(limit:10){ id items{ product{ name } } } }`);
  return `query($id: ID!){ ${parts.join(" ")} }`;
}

// Zapytanie, którego realna praca rośnie z `limit`: od listy zamówień (Query.orders zwraca L zamówień), przez
// Order.user, do User.orders(limit:L). Każdy użytkownik ma w seedzie dokładnie 5 zamówień, więc zagnieżdżenie od
// User.orders (jak wcześniej) dawało zawsze 5 wyników niezależnie od L. Tu: L zamówień -> L użytkowników -> 5L zamówień
// -> 20L pozycji. Koszt w modelu complexity.go: patrz TestCalibration (app/internal/graphql/complexity_test.go).
// Bez zmiennych: seria_d wysyła variables {id}, a zadeklarowana nieużyta zmienna jest błędem walidacji GraphQL.
export function bigLimitQuery(limit) {
  return `query{ orders(limit:${limit}){ id user{ orders(limit:${limit}){ id items{ quantity product{ id name } } } } } }`;
}

// ---- seria C: ten sam ładunek logiczny (zamówienie z N pozycjami) w każdym interfejsie ----
// N = floor(SIZE / średni rozmiar pozycji w JSON), wspólne dla REST, GraphQL i gRPC: wszystkie interfejsy dostają
// tę samą liczbę pozycji (ta sama praca serwera i bazy), a rozmiar na drucie różni się formatem (JSON vs protobuf).
// Rzeczywisty rozmiar i liczbę pozycji zapisują metryki attack_payload_bytes / attack_payload_items (payloadMetrics).
// Ładunek jest budowany raz na VU (cache), żeby nie obciążać generatora.
const payloadCache = {};

function itemsOf(n) {
  const items = [];
  for (let i = 0; i < n; i++) items.push({ productId: 1 + (i % 5000), quantity: 1 });
  return items;
}

const JSON_BYTES_PER_ITEM = JSON.stringify(itemsOf(1000)).length / 1000; // średni rozmiar pozycji w JSON
export function itemsForSize(bytes) {
  return Math.max(1, Math.floor(bytes / JSON_BYTES_PER_ITEM));
}

function cached(key, build) {
  if (!(key in payloadCache)) payloadCache[key] = build();
  return payloadCache[key];
}

export function bigOrderRest(bytes) {
  return cached(`rest:${bytes}`, () => JSON.stringify({ userId: 1, items: itemsOf(itemsForSize(bytes)) }));
}

export function bigOrderGraphql(bytes) {
  return cached(`gql:${bytes}`, () => JSON.stringify({ query: GQL.O4, variables: { in: { userId: 1, items: itemsOf(itemsForSize(bytes)) } } }));
}

export function bigOrderGrpc(bytes) {
  return cached(`grpc:${bytes}`, () => ({
    user_id: 1,
    items: itemsOf(itemsForSize(bytes)).map((i) => ({ product_id: i.productId, quantity: i.quantity })),
  }));
}

// Dokładny rozmiar zakodowanej wiadomości CreateOrderRequest (protobuf, proto3): k6 go nie raportuje.
// Pola: user_id = 1 (int64, varint), items = 2 (repeated NewItem, length-delimited); NewItem: product_id = 1, quantity = 2.
// Wartości 0 w proto3 nie są kodowane. Tagi pól < 16 zajmują 1 bajt.
function varintLen(v) {
  let n = 1;
  while (v >= 128) { v = Math.floor(v / 128); n++; }
  return n;
}
export function protoSizeCreateOrder(msg) {
  let size = msg.user_id ? 1 + varintLen(msg.user_id) : 0;
  for (const it of msg.items) {
    const inner = (it.product_id ? 1 + varintLen(it.product_id) : 0) + (it.quantity ? 1 + varintLen(it.quantity) : 0);
    size += 1 + varintLen(inner) + inner;
  }
  return size;
}

// ---- klient gRPC ----
export const grpcClient = new grpc.Client();
grpcClient.load(["./lib"], "service.proto");

let connected = false;
export function grpcConnect() {
  if (!connected) {
    grpcClient.connect(HOST, { plaintext: !TLS, timeout: "10s", reflect: false });
    connected = true;
  }
}

export function grpcMeta(token) { return token ? { metadata: { authorization: `Bearer ${token}` } } : {}; }
