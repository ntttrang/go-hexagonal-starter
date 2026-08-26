import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';

// Endpoint: POST /api/v1/auth/register
// Generates a unique email per iteration using k6's execution context vars.
// Requires the DB to be reachable (registration persists a user).

const errorRate = new Rate('register_errors');
const created = new Counter('users_created');

export const options = {
  stages: [
    { duration: '15s', target: 10 },   // ramp up to 10 VUs
    { duration: '30s', target: 10 },   // hold
    { duration: '15s', target: 0 },    // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    register_errors: ['rate<0.05'],
    http_req_failed: ['rate<0.05'],    // k6 built-in: non-2xx/3xx
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8085';

export default function () {
  // __VU and __ITER are k6 built-ins -> unique value per virtual user & iteration
  const email = `u${__VU}_${__ITER}@loadtest.local`;

  const res = http.post(
    `${BASE}/api/v1/auth/register`,
    JSON.stringify({
      email: email,
      name: 'Loadtest User',
      password: 'secret123',
    }),
    { headers: { 'Content-Type': 'application/json' } },
  );

  const ok = check(res, {
    'status is 201': (r) => r.status === 201,
    'has user id': (r) => r.json('id') !== undefined,
  });

  if (ok) created.add(1);
  errorRate.add(!ok);
  sleep(0.5);
}
