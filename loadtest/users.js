import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Authenticated read path:
//   setup()  -> register a user, then login to obtain a JWT
//   default  -> GET /api/v1/users with that token (exercises DB + auth middleware)
// Requires the DB to be up and migrations applied (registration persists a user).

const errorRate = new Rate('users_errors');

export const options = {
  vus: 10,
  duration: '30s',
  thresholds: {
    http_req_duration: ['p(95)<300'],
    users_errors: ['rate<0.01'],
    http_req_failed: ['rate<0.01'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8085';

export function setup() {
  // Unique email so repeated runs don't collide on the unique constraint.
  const email = `bench-${Date.now()}@loadtest.local`;
  const password = 'secret123';

  const reg = http.post(
    `${BASE}/api/v1/auth/register`,
    JSON.stringify({ email, name: 'Bench User', password }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  check(reg, { 'registered (201)': (r) => r.status === 201 });

  const loginRes = http.post(
    `${BASE}/api/v1/auth/login`,
    JSON.stringify({ email, password }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  check(loginRes, { 'login ok (200)': (r) => r.status === 200 });

  const token = loginRes.json('token');
  if (!token) {
    throw new Error(`setup failed: login returned no token (register=${reg.status}, login=${loginRes.status})`);
  }
  return { token };
}

export default function (data) {
  const res = http.get(`${BASE}/api/v1/users`, {
    headers: { Authorization: `Bearer ${data.token}` },
  });

  const ok = check(res, {
    'status is 200': (r) => r.status === 200,
    'returns a list': (r) => Array.isArray(r.json()),
  });
  errorRate.add(!ok);
  sleep(0.5);
}
