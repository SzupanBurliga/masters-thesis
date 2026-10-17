// Wykonanie operacji O1-O4 dla wybranego interfejsu; zwraca true dla poprawnej odpowiedzi.
import http from "k6/http";
import grpc from "k6/net/grpc";
import { check } from "k6";
import { IFACE, BASE, GQL, headers, pickId, newOrderBody, grpcClient, grpcConnect, grpcMeta } from "./common.js";

export function doOp(op, token, tags) {
  const id = pickId();
  const params = { headers: headers(token), tags: tags || {} };
  if (IFACE === "rest") {
    let r;
    if (op === "O1") r = http.get(`${BASE}/users/${id}`, params);
    else if (op === "O2") r = http.get(`${BASE}/orders?status=PAID&limit=20&offset=0`, params);
    else if (op === "O3") r = http.get(`${BASE}/users/${id}/orders?expand=items.product&limit=10`, params);
    else r = http.post(`${BASE}/orders`, JSON.stringify(newOrderBody(id, 3)), params);
    return check(r, { ok: (x) => x.status === 200 || x.status === 201 });
  }
  if (IFACE === "graphql") {
    const body = op === "O4" ? { query: GQL.O4, variables: { in: newOrderBody(id, 3) } } : { query: GQL[op], variables: { id } };
    const r = http.post(`${BASE}/graphql`, JSON.stringify(body), params);
    return check(r, { ok: (x) => x.status === 200 && !x.body.includes('"errors"') });
  }
  grpcConnect();
  const meta = { ...grpcMeta(token), tags: tags || {} };
  let r;
  if (op === "O1") r = grpcClient.invoke("shop.v1.Shop/GetUser", { id }, meta);
  else if (op === "O2") r = grpcClient.invoke("shop.v1.Shop/ListOrders", { status: "PAID", limit: 20, offset: 0 }, meta);
  else if (op === "O3") r = grpcClient.invoke("shop.v1.Shop/GetUserWithOrders", { id, limit: 10 }, meta);
  else r = grpcClient.invoke("shop.v1.Shop/CreateOrder", { user_id: id, items: newOrderBody(id, 3).items.map((i) => ({ product_id: i.productId, quantity: i.quantity })) }, meta);
  return check(r, { ok: (x) => x && x.status === grpc.StatusOK });
}
