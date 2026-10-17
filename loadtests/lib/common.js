// Wspólne parametry i generatory dla skryptów k6 (wszystko deterministyczne: bez Math.random).
import { SharedArray } from "k6/data";
import grpc from "k6/net/grpc";

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

// Zapytanie o kosztowej strukturze rosnącej z `limit` (cykl Order.user -> User.orders).
// Koszt (model z complexity.go): L=10 -> 321, L=25 -> 1926, L=50 -> 7601 (serwer ucina limit do 50).
export function bigLimitQuery(limit) {
  return `query($id: ID!){ user(id:$id){ orders(limit:${limit}){ id user{ orders(limit:${limit}){ id items{ quantity product{ id name } } } } } } }`;
}

// ---- seria C: ten sam ładunek logiczny (zamówienie z N pozycjami) w każdym interfejsie ----
// N dobrane tak, by rozmiar wiadomości NA DRUCIE ≈ SIZE bajtów. Wszystkie interfejsy wykonują
// N wstawień do bazy, więc różni je tylko koszt parsowania i działanie mechanizmów (limit rozmiaru,
// długość tablicy). Ładunek jest budowany raz na VU (cache), żeby nie obciążać generatora.
const GRPC_BYTES_PER_ITEM = 7; // protobuf: tag+len (2) + product_id (3) + quantity (2)
const payloadCache = {};

function itemsOf(n) {
  const items = [];
  for (let i = 0; i < n; i++) items.push({ productId: 1 + (i % 5000), quantity: 1 });
  return items;
}

function jsonItemsFor(bytes) {
  const sample = JSON.stringify(itemsOf(1000)).length / 1000; // średni rozmiar pozycji w JSON
  return itemsOf(Math.max(1, Math.floor(bytes / sample)));
}

function cached(key, build) {
  if (!(key in payloadCache)) payloadCache[key] = build();
  return payloadCache[key];
}

export function bigOrderRest(bytes) {
  return cached(`rest:${bytes}`, () => JSON.stringify({ userId: 1, items: jsonItemsFor(bytes) }));
}

export function bigOrderGraphql(bytes) {
  return cached(`gql:${bytes}`, () => JSON.stringify({ query: GQL.O4, variables: { in: { userId: 1, items: jsonItemsFor(bytes) } } }));
}

export function bigOrderGrpc(bytes) {
  return cached(`grpc:${bytes}`, () => ({
    user_id: 1,
    items: itemsOf(Math.max(1, Math.floor(bytes / GRPC_BYTES_PER_ITEM))).map((i) => ({ product_id: i.productId, quantity: i.quantity })),
  }));
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
