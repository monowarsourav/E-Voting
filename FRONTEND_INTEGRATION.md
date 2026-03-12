# Frontend Integration Guide

This guide explains how to integrate a frontend application with the CovertVote E-Voting API.

## 1. Starting the Backend Services

### Option A: Run All Services Separately
```bash
# Terminal 1: Start SA² Aggregator Server A
make run-aggregator-a
# Runs on: http://localhost:8081

# Terminal 2: Start SA² Aggregator Server B
make run-aggregator-b
# Runs on: http://localhost:8082

# Terminal 3: Start API Server
make run-api
# Runs on: http://localhost:8080
```

### Option B: Use Docker Compose (if you have sufficient resources)
```bash
docker-compose up
```

## 2. API Base URL
```
Base URL: http://localhost:8080
API Version: /api/v1
```

## 3. Available API Endpoints

### Public Endpoints (No Authentication Required)

#### Health Check
```
GET /health
Response: {
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 12345
}
```

#### Register Voter
```
POST /api/v1/register
Content-Type: application/json

Request Body:
{
  "voter_id": "NID12345",
  "fingerprint_data": [base64_encoded_fingerprint],
  "liveness_data": [base64_encoded_liveness],
  "biographic_data": "optional metadata",
  "eligibility_proof": "optional proof"
}

Response (201):
{
  "voter_id": "generated_voter_id",
  "public_key": "hex_encoded_public_key",
  "smdc_public_credential": "hex_encoded_credential",
  "merkle_root": "hex_encoded_merkle_root",
  "registration_time": 1234567890,
  "message": "Registration successful"
}

Sets Cookie: session_token (valid for 24 hours)
```

#### Get All Elections
```
GET /api/v1/elections

Response:
{
  "elections": [
    {
      "election_id": "election001",
      "title": "Presidential Election 2026",
      "description": "Annual presidential election",
      "candidates": [
        {
          "id": 1,
          "name": "Candidate A",
          "description": "Experienced leader",
          "party": "Party 1"
        }
      ],
      "start_time": 1234567890,
      "end_time": 1234567890,
      "is_active": true,
      "total_votes": 0
    }
  ],
  "total": 1
}
```

#### Get Single Election
```
GET /api/v1/elections/:id

Response:
{
  "election_id": "election001",
  "title": "Presidential Election 2026",
  "candidates": [...],
  "start_time": 1234567890,
  "end_time": 1234567890,
  "is_active": true
}
```

#### Get Election Results
```
GET /api/v1/results/:electionId

Response:
{
  "election_id": "election001",
  "candidate_tallies": {
    "1": 150,
    "2": 200
  },
  "total_votes": 350,
  "tally_time": 1234567890,
  "verified": true
}
```

#### Get Vote Count
```
GET /api/v1/vote-count

Response:
{
  "total_votes": 350,
  "timestamp": 1234567890
}
```

### Authenticated Endpoints (Requires session_token Cookie)

#### Get Voter Info
```
GET /api/v1/voter/:id
Cookie: session_token=your_session_token

Response:
{
  "voter_id": "voter123",
  "registered": true,
  "registration_time": 1234567890,
  "has_voted": false
}
```

#### Cast Vote
```
POST /api/v1/vote
Cookie: session_token=your_session_token
Content-Type: application/json

Request:
{
  "voter_id": "voter123",
  "election_id": "election001",
  "candidate_id": 1,
  "smdc_slot_index": 0,
  "auth_token": "session_token_value"
}

Response (200):
{
  "receipt_id": "receipt_xyz",
  "voter_id": "voter123",
  "election_id": "election001",
  "timestamp": 1234567890,
  "blockchain_tx_id": "tx_hash",
  "key_image": "hex_key_image",
  "message": "Vote cast successfully"
}
```

#### Verify Vote
```
POST /api/v1/verify-vote
Cookie: session_token=your_session_token
Content-Type: application/json

Request:
{
  "receipt_id": "receipt_xyz",
  "voter_id": "voter123"
}

Response:
{
  "valid": true,
  "election_id": "election001",
  "timestamp": 1234567890,
  "blockchain_tx_id": "tx_hash",
  "message": "Vote verified"
}
```

### Admin Endpoints (Requires Admin Token)

Set admin token in `.env`:
```
ADMIN_TOKEN=your-admin-token-here
```

#### Create Election
```
POST /api/v1/admin/elections
Content-Type: application/json

Request:
{
  "title": "New Election",
  "description": "Description",
  "candidates": [
    {
      "id": 1,
      "name": "Candidate A",
      "description": "Description",
      "party": "Party 1"
    },
    {
      "id": 2,
      "name": "Candidate B",
      "description": "Description",
      "party": "Party 2"
    }
  ],
  "start_time": 1234567890,
  "end_time": 1234567890,
  "admin_token": "your-admin-token"
}

Response (201):
{
  "election_id": "generated_id",
  "title": "New Election",
  "candidates": [...],
  "start_time": 1234567890,
  "end_time": 1234567890,
  "is_active": true,
  "message": "Election created successfully"
}
```

#### Tally Votes
```
POST /api/v1/admin/tally
Content-Type: application/json

Request:
{
  "election_id": "election001",
  "admin_token": "your-admin-token"
}

Response:
{
  "election_id": "election001",
  "candidate_tallies": {
    "1": 150,
    "2": 200
  },
  "total_votes": 350,
  "tally_time": 1234567890,
  "verified": true
}
```

#### Get All Voters (Admin)
```
GET /api/v1/admin/voters
Header: X-Admin-Token: your-admin-token

Response:
{
  "total_voters": 100,
  "merkle_root": "hex_merkle_root"
}
```

## 4. Frontend Implementation Examples

### A. Using Vanilla JavaScript (Fetch API)

```javascript
// config.js
const API_BASE_URL = 'http://localhost:8080';
const API_VERSION = '/api/v1';

// Helper function for API calls
async function apiCall(endpoint, options = {}) {
  const url = `${API_BASE_URL}${endpoint}`;
  const defaultOptions = {
    headers: {
      'Content-Type': 'application/json',
    },
    credentials: 'include', // Important: Include cookies
  };

  const response = await fetch(url, { ...defaultOptions, ...options });
  const data = await response.json();

  if (!response.ok) {
    throw new Error(data.message || 'API request failed');
  }

  return data;
}

// Get all elections
async function getElections() {
  try {
    const data = await apiCall(`${API_VERSION}/elections`);
    console.log('Elections:', data.elections);
    return data.elections;
  } catch (error) {
    console.error('Error fetching elections:', error);
  }
}

// Register voter
async function registerVoter(voterData) {
  try {
    const data = await apiCall(`${API_VERSION}/register`, {
      method: 'POST',
      body: JSON.stringify(voterData),
    });
    console.log('Registration successful:', data);
    return data;
  } catch (error) {
    console.error('Registration failed:', error);
  }
}

// Cast vote
async function castVote(voteData) {
  try {
    const data = await apiCall(`${API_VERSION}/vote`, {
      method: 'POST',
      body: JSON.stringify(voteData),
    });
    console.log('Vote cast successfully:', data);
    return data;
  } catch (error) {
    console.error('Vote casting failed:', error);
  }
}

// Get results
async function getResults(electionId) {
  try {
    const data = await apiCall(`${API_VERSION}/results/${electionId}`);
    console.log('Results:', data);
    return data;
  } catch (error) {
    console.error('Error fetching results:', error);
  }
}
```

### B. Using React with Axios

```javascript
// src/api/votingApi.js
import axios from 'axios';

const API = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  withCredentials: true, // Important: Include cookies
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add response interceptor for error handling
API.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error.response?.data || error.message);
    return Promise.reject(error);
  }
);

export const votingApi = {
  // Elections
  getElections: () => API.get('/elections'),
  getElection: (id) => API.get(`/elections/${id}`),

  // Registration
  registerVoter: (data) => API.post('/register', data),
  verifyEligibility: (data) => API.post('/verify-eligibility', data),

  // Voting
  castVote: (data) => API.post('/vote', data),
  verifyVote: (data) => API.post('/verify-vote', data),

  // Results
  getResults: (electionId) => API.get(`/results/${electionId}`),
  getVoteCount: () => API.get('/vote-count'),

  // Voter Info
  getVoterInfo: (voterId) => API.get(`/voter/${voterId}`),

  // Admin
  createElection: (data) => API.post('/admin/elections', data),
  tallyVotes: (data) => API.post('/admin/tally', data),
  getAllVoters: (adminToken) =>
    API.get('/admin/voters', {
      headers: { 'X-Admin-Token': adminToken }
    }),
};

export default votingApi;
```

```javascript
// src/components/ElectionList.jsx
import React, { useState, useEffect } from 'react';
import votingApi from '../api/votingApi';

function ElectionList() {
  const [elections, setElections] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    loadElections();
  }, []);

  const loadElections = async () => {
    try {
      const response = await votingApi.getElections();
      setElections(response.data.elections);
      setLoading(false);
    } catch (err) {
      setError(err.message);
      setLoading(false);
    }
  };

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <div>
      <h2>Active Elections</h2>
      {elections.map((election) => (
        <div key={election.election_id} className="election-card">
          <h3>{election.title}</h3>
          <p>{election.description}</p>
          <p>Status: {election.is_active ? 'Active' : 'Closed'}</p>
          <p>Total Votes: {election.total_votes}</p>
          <h4>Candidates:</h4>
          <ul>
            {election.candidates.map((candidate) => (
              <li key={candidate.id}>
                {candidate.name} - {candidate.party}
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}

export default ElectionList;
```

### C. Using Vue.js

```javascript
// src/services/api.js
import axios from 'axios';

const apiClient = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
});

export default {
  getElections() {
    return apiClient.get('/elections');
  },
  registerVoter(voterData) {
    return apiClient.post('/register', voterData);
  },
  castVote(voteData) {
    return apiClient.post('/vote', voteData);
  },
  getResults(electionId) {
    return apiClient.get(`/results/${electionId}`);
  },
};
```

```vue
<!-- src/components/Elections.vue -->
<template>
  <div class="elections">
    <h2>Active Elections</h2>
    <div v-if="loading">Loading...</div>
    <div v-else-if="error">{{ error }}</div>
    <div v-else>
      <div v-for="election in elections" :key="election.election_id" class="election">
        <h3>{{ election.title }}</h3>
        <p>{{ election.description }}</p>
        <ul>
          <li v-for="candidate in election.candidates" :key="candidate.id">
            {{ candidate.name }} - {{ candidate.party }}
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script>
import api from '@/services/api';

export default {
  data() {
    return {
      elections: [],
      loading: true,
      error: null,
    };
  },
  async mounted() {
    try {
      const response = await api.getElections();
      this.elections = response.data.elections;
    } catch (error) {
      this.error = error.message;
    } finally {
      this.loading = false;
    }
  },
};
</script>
```

## 5. Authentication Flow

### Session-Based Authentication (Used for Voters)

1. **Registration**: User registers via POST `/api/v1/register`
2. **Server Response**: Returns `session_token` cookie (24-hour validity)
3. **Subsequent Requests**: Browser automatically sends cookie
4. **Session Validation**: Middleware validates session on protected routes

```javascript
// After successful registration
async function registerAndLogin(voterData) {
  const response = await fetch('http://localhost:8080/api/v1/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include', // Important!
    body: JSON.stringify(voterData),
  });

  const data = await response.json();
  // Cookie is automatically stored by browser
  // Now you can make authenticated requests

  return data;
}

// Making authenticated request
async function getMyVoterInfo(voterId) {
  const response = await fetch(`http://localhost:8080/api/v1/voter/${voterId}`, {
    credentials: 'include', // Sends cookie automatically
  });

  return response.json();
}
```

### Admin Token Authentication

Admin endpoints require the `ADMIN_TOKEN` from `.env` file:

```javascript
const ADMIN_TOKEN = 'your-admin-token-here'; // From .env

async function createElection(electionData) {
  const response = await fetch('http://localhost:8080/api/v1/admin/elections', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      ...electionData,
      admin_token: ADMIN_TOKEN,
    }),
  });

  return response.json();
}
```

## 6. CORS Configuration

The API already has CORS enabled in `api/middleware/cors.go`:
- Allows all origins in development
- Allows credentials (cookies)
- Allows all common HTTP methods
- Allows all headers

If you need to restrict origins in production, update the middleware.

## 7. Error Handling

All errors follow this format:
```json
{
  "error": "error_code",
  "code": 400,
  "message": "Human-readable error message"
}
```

Common error codes:
- `400`: Bad Request (invalid data)
- `401`: Unauthorized (missing/invalid session)
- `403`: Forbidden (liveness failed, already voted)
- `404`: Not Found (voter/election not found)
- `429`: Too Many Requests (rate limit exceeded)
- `500`: Internal Server Error

## 8. Rate Limiting

The API has built-in rate limiting:
- **Public routes**: Standard rate limit (moderate)
- **Authenticated routes**: Strict rate limit (more restrictive)
- **Admin routes**: Strict rate limit

If rate limited, you'll receive a `429 Too Many Requests` response.

## 9. Testing the API

### Using cURL

```bash
# Health check
curl http://localhost:8080/health

# Get elections
curl http://localhost:8080/api/v1/elections

# Get results
curl http://localhost:8080/api/v1/results/election001

# Register voter (simplified - you need actual biometric data)
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "voter_id": "NID12345",
    "fingerprint_data": "base64data",
    "liveness_data": "base64data"
  }'
```

### Using Postman

1. Import the API endpoints as a collection
2. Set base URL: `http://localhost:8080`
3. Enable "Send cookies" in settings
4. Test each endpoint with sample data

## 10. Next Steps

### For Development:
1. Start the backend services (see Section 1)
2. Create your frontend app (React/Vue/Angular/etc.)
3. Install axios or use fetch API
4. Implement the API calls (see Section 4)
5. Handle authentication (see Section 5)
6. Test with sample data

### For Production:
1. Change API base URL to production server
2. Update CORS allowed origins
3. Use HTTPS for all requests
4. Implement proper error handling
5. Add retry logic for failed requests
6. Implement request/response logging
7. Add loading states and user feedback

## 11. Sample Complete Workflow

```javascript
// 1. Get available elections
const elections = await votingApi.getElections();
console.log('Available elections:', elections.data);

// 2. Register voter
const registration = await votingApi.registerVoter({
  voter_id: 'NID12345',
  fingerprint_data: fingerprintBytes,
  liveness_data: livenessBytes,
});
console.log('Registered:', registration.data);
// Session cookie is now set automatically

// 3. Cast vote (using the session)
const vote = await votingApi.castVote({
  voter_id: registration.data.voter_id,
  election_id: 'election001',
  candidate_id: 1,
  smdc_slot_index: 0,
  auth_token: getCookie('session_token'),
});
console.log('Vote receipt:', vote.data);

// 4. Verify vote
const verification = await votingApi.verifyVote({
  receipt_id: vote.data.receipt_id,
  voter_id: registration.data.voter_id,
});
console.log('Vote verified:', verification.data);

// 5. Get results (after admin tallies)
const results = await votingApi.getResults('election001');
console.log('Election results:', results.data);
```

## 12. Important Notes

- **Biometric Data**: In production, you'll need actual fingerprint scanner integration
- **Session Management**: Sessions expire after 24 hours
- **SMDC Slot Index**: Use slot 0 for the real vote (slots 1-4 are fake/coercion-resistant)
- **Database**: All data is persisted in SQLite at `./data/covertvote.db`
- **Migrations**: Automatically run on server startup

## Support

For issues or questions:
- Check the API logs in the terminal
- Verify backend services are running
- Check CORS settings if requests are blocked
- Ensure cookies are enabled in your browser/client
