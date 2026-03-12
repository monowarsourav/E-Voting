# CovertVote: সম্পূর্ণ গবেষণা বিশ্লেষণ

## সূচিপত্র

1. [প্রজেক্ট সংক্ষিপ্ত বিবরণ](#১-প্রজেক্ট-সংক্ষিপ্ত-বিবরণ)
2. [সিস্টেম আর্কিটেকচার](#২-সিস্টেম-আর্কিটেকচার)
3. [সম্পূর্ণ কার্যপ্রবাহ (Working Flow)](#৩-সম্পূর্ণ-কার্যপ্রবাহ)
4. [ক্রিপ্টোগ্রাফিক প্রোটোকল বিশ্লেষণ](#৪-ক্রিপ্টোগ্রাফিক-প্রোটোকল-বিশ্লেষণ)
5. [শক্তি (Strengths)](#৫-শক্তি-strengths)
6. [দুর্বলতা (Weaknesses)](#৬-দুর্বলতা-weaknesses)
7. [সুবিধা (Pros)](#৭-সুবিধা-pros)
8. [অসুবিধা (Cons)](#৮-অসুবিধা-cons)
9. [বিদ্যমান Blockchain ই-ভোটিং সিস্টেমের তুলনা](#৯-বিদ্যমান-সিস্টেমের-তুলনা)
10. [একাডেমিক পেপারের সাথে তুলনা](#১০-একাডেমিক-পেপারের-সাথে-তুলনা)
11. [বাস্তবায়িত সিস্টেমের সাথে তুলনা](#১১-বাস্তবায়িত-সিস্টেমের-সাথে-তুলনা)
12. [গবেষণায় CovertVote-এর অবদান](#১২-গবেষণায়-অবদান)
13. [সীমাবদ্ধতা ও ভবিষ্যৎ কাজ](#১৩-সীমাবদ্ধতা-ও-ভবিষ্যৎ-কাজ)
14. [উপসংহার](#১৪-উপসংহার)
15. [তথ্যসূত্র](#১৫-তথ্যসূত্র)

---

## ১. প্রজেক্ট সংক্ষিপ্ত বিবরণ

**CovertVote** হলো Go ভাষায় তৈরি একটি Blockchain-ভিত্তিক ই-ভোটিং সিস্টেম যা উন্নত ক্রিপ্টোগ্রাফিক প্রোটোকল ব্যবহার করে **গোপনীয়তা**, **বলপ্রয়োগ-প্রতিরোধ**, এবং **যাচাইযোগ্যতা** একসাথে নিশ্চিত করে।

**মূল উদ্ভাবন**: এই সিস্টেম ৭টি ক্রিপ্টোগ্রাফিক প্রোটোকল একত্রিত করে একটি সম্পূর্ণ ভোটিং সমাধান তৈরি করেছে — যা বিদ্যমান কোনো সিস্টেমে একসাথে পাওয়া যায় না।

| বিষয় | বিবরণ |
|---|---|
| ভাষা | Go 1.24.0 |
| ফ্রেমওয়ার্ক | Gin 1.11.0 (REST API) |
| Blockchain | Hyperledger Fabric (Chaincode) |
| কোড | ~৬,০৮০ LOC (সোর্স) + ~১,৫০০ LOC (টেস্ট) |
| টেস্ট | ৫১টি, ১০০% পাস রেট |
| কভারেজ | ৬৮.৫% গড় |

---

## ২. সিস্টেম আর্কিটেকচার

### ২.১ স্তরভিত্তিক আর্কিটেকচার (Layered Architecture)

```
┌─────────────────────────────────────────────────────┐
│                  Client Layer                        │
│         (ভোটার ইন্টারফেস / মোবাইল / ওয়েব)          │
└───────────────────────┬─────────────────────────────┘
                        │ HTTPS/REST
┌───────────────────────▼─────────────────────────────┐
│               API Layer (Gin REST)                   │
│  ১৫+ Endpoints: /auth, /vote, /election, /tally     │
│  Rate Limiting | CORS | JWT Authentication           │
└───────────────────────┬─────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────┐
│            Computation Layer                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐  │
│  │ Paillier │ │ Pedersen  │ │   ZKP    │ │  Ring  │  │
│  │  HE      │ │Commitment│ │Σ-Protocol│ │  Sig   │  │
│  └──────────┘ └──────────┘ └──────────┘ └────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐            │
│  │  SMDC    │ │Biometric │ │ Merkle   │            │
│  │Credential│ │  Auth    │ │  Tree    │            │
│  └──────────┘ └──────────┘ └──────────┘            │
└───────────────────────┬─────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────┐
│            Privacy Layer (SA²)                       │
│  ┌─────────────┐          ┌─────────────┐           │
│  │  Server A   │◄────────►│  Server B   │           │
│  │  (Leader)   │  Mask    │  (Helper)   │           │
│  │ share + mask│Cancellat.│ share + mask│           │
│  └──────┬──────┘          └──────┬──────┘           │
│         └──────────┬─────────────┘                  │
│                    │ Combined Result                 │
└────────────────────┬────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────┐
│            Storage Layer                             │
│  ┌──────────────────┐  ┌─────────────────────────┐  │
│  │  Hyperledger     │  │  SQLite                 │  │
│  │  Fabric          │  │  (ভোটার ডেটা, সেশন)      │  │
│  │  (ভোট রেকর্ড)    │  │                         │  │
│  └──────────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

### ২.২ পোস্ট-কোয়ান্টাম স্তর (অতিরিক্ত)

```
┌─────────────────────────────────────┐
│   Post-Quantum Layer               │
│   Kyber768 (CIRCL Library)         │
│   Hybrid Encryption                │
│   ভবিষ্যতের কোয়ান্টাম আক্রমণ থেকে │
│   সুরক্ষা                          │
└─────────────────────────────────────┘
```

---

## ৩. সম্পূর্ণ কার্যপ্রবাহ

### ৩.১ ভোটার নিবন্ধন প্রবাহ

```
ভোটার → আঙুলের ছাপ → SHA-3 হ্যাশ → Liveness Detection
                                          │
                              ├── সফল: Merkle Tree-তে যোগ
                              └── ব্যর্থ: প্রত্যাখ্যান
```

**ধাপসমূহ:**
1. ভোটার আঙুলের ছাপ জমা দেয়
2. SHA-3 হ্যাশিং এর মাধ্যমে বায়োমেট্রিক টেমপ্লেট তৈরি
3. Liveness Detection যাচাই (স্পুফিং প্রতিরোধ)
4. Merkle Tree-তে ভোটার ID যোগ (O(log n) যাচাই)
5. ভোটারকে SMDC credential প্রদান

### ৩.২ ভোট কাস্টিং প্রবাহ (১৫-ধাপ পাইপলাইন)

```
ধাপ ১: বায়োমেট্রিক প্রমাণীকরণ
    │   আঙুলের ছাপ → SHA-3 → টেমপ্লেট ম্যাচিং
    ▼
ধাপ ২: Merkle Proof যাচাই
    │   ভোটারের যোগ্যতা O(log n) সময়ে প্রমাণ
    ▼
ধাপ ৩: SMDC Credential নির্বাচন
    │   k=5 slots থেকে সঠিক credential ব্যবহার
    │   (চাপের মুখে নকল credential দেখানো সম্ভব)
    ▼
ধাপ ৪: ভোট নির্বাচন
    │   প্রার্থী বাছাই (w ∈ {0, 1} প্রতি প্রার্থীতে)
    ▼
ধাপ ৫: Paillier Encryption
    │   ভোট এনক্রিপ্ট: E(vote) = g^vote × r^n mod n²
    │   (2048-bit key)
    ▼
ধাপ ৬: Pedersen Commitment
    │   C = g^v × h^r mod p (ভোটের প্রতিশ্রুতি)
    ▼
ধাপ ৭: ZKP তৈরি — Binary Proof
    │   প্রমাণ যে w ∈ {0, 1} (কোনো অবৈধ মান নেই)
    ▼
ধাপ ৮: ZKP তৈরি — Sum Proof
    │   প্রমাণ যে Σw = 1 (ঠিক ১ জন প্রার্থীকে ভোট)
    ▼
ধাপ ৯: Fiat-Shamir Transformation
    │   Interactive proof → Non-interactive রূপান্তর
    ▼
ধাপ ১০: Ring Signature
    │   ১০০-সদস্যের ring-এ বেনামী স্বাক্ষর
    │   Key Image দিয়ে ডাবল-ভোট শনাক্তকরণ
    ▼
ধাপ ১১: SA² Share তৈরি
    │   ভোট = share_A + share_B (additive secret sharing)
    │   প্রতিটি share-এ random mask যোগ
    ▼
ধাপ ১২: Server A-তে Share পাঠানো
    │   share_A + mask_A
    ▼
ধাপ ১৩: Server B-তে Share পাঠানো
    │   share_B + mask_B
    │   (mask_A + mask_B = 0, তাই mask বাতিল হয়)
    ▼
ধাপ ১৪: Blockchain-এ রেকর্ড
    │   Hyperledger Fabric chaincode-এ ভোট সংরক্ষণ
    ▼
ধাপ ১৫: রসিদ প্রদান
    │   ভোটারকে যাচাইযোগ্য রসিদ দেওয়া
```

### ৩.৩ ভোট গণনা প্রবাহ

```
SA² Server A         SA² Server B
    │                     │
    │  Homomorphic        │  Homomorphic
    │  Addition           │  Addition
    │  E(v₁)×E(v₂)×...   │  E(v₁)×E(v₂)×...
    │  = E(Σvᵢ)          │  = E(Σvᵢ)
    ▼                     ▼
    └────────┬────────────┘
             │ Threshold Decryption (2-of-2)
             ▼
      চূড়ান্ত ফলাফল প্রকাশ
      (কোনো একক সার্ভার একা ফলাফল দেখতে পারে না)
```

---

## ৪. ক্রিপ্টোগ্রাফিক প্রোটোকল বিশ্লেষণ

### ৪.১ Paillier Homomorphic Encryption

| বিষয় | বিবরণ |
|---|---|
| ধরন | Additively Homomorphic |
| Key Size | 2048-bit |
| মূল বৈশিষ্ট্য | E(a) × E(b) = E(a+b) — এনক্রিপ্টেড অবস্থায় ভোট যোগ |
| নিরাপত্তা ভিত্তি | Decisional Composite Residuosity Assumption (DCRA) |
| ব্যবহার | ভোট এনক্রিপশন ও গণনা |

**কেন Paillier?** ভোটিং-এ শুধু **যোগ (addition)** প্রয়োজন — কোন ভোটার কোন প্রার্থীকে ভোট দিয়েছে তা না জেনেই মোট গণনা সম্ভব। ElGamal-ও homomorphic কিন্তু multiplicatively — Paillier-এর additive property ভোটিং-এর জন্য বেশি উপযুক্ত।

### ৪.২ SMDC (Self-Masking Deniable Credentials)

| বিষয় | বিবরণ |
|---|---|
| Slot সংখ্যা | k = 5 (১টি আসল + ৪টি নকল) |
| উদ্দেশ্য | বলপ্রয়োগ-প্রতিরোধ (Coercion Resistance) |
| নীতি | Deniable Authentication |

**কীভাবে কাজ করে:**
- ভোটার ৫টি credential পায় — শুধু ১টি আসল
- বলপ্রয়োগকারী জানে না কোনটি আসল
- চাপের মুখে ভোটার নকল credential দেখাতে পারে
- নকল credential-এ দেওয়া ভোট চূড়ান্ত গণনায় বাদ যায়
- কিন্তু বাইরে থেকে আসল আর নকল আলাদা করা **গাণিতিকভাবে অসম্ভব**

### ৪.৩ SA² (Samplable Anonymous Aggregation)

| বিষয় | বিবরণ |
|---|---|
| মডেল | 2-Server (Leader + Helper) |
| ভিত্তি | Prio Protocol (Apple Research, ACM CCS 2024) |
| Mask | mask_A + mask_B = 0 (cancellation) |

**কেন SA²?** Apple-এর গবেষণা থেকে উদ্ভূত এই প্রোটোকল Federated Data Analysis-এ ব্যবহৃত হয়। CovertVote এটিকে ভোটিং-এ রূপান্তরিত করেছে। দুটি সার্ভারের মধ্যে যেকোনো একটি সৎ থাকলেই সম্পূর্ণ গোপনীয়তা বজায় থাকে।

### ৪.৪ Zero-Knowledge Proofs (Σ-Protocol)

| প্রমাণের ধরন | কাজ |
|---|---|
| Binary Proof | ভোট w ∈ {0, 1} (বৈধ মান) |
| Sum Proof | Σw = 1 (ঠিক একটি প্রার্থীকে ভোট) |
| Fiat-Shamir | Interactive → Non-interactive রূপান্তর |

### ৪.৫ Linkable Ring Signatures

| বিষয় | বিবরণ |
|---|---|
| Ring Size | ১০০ সদস্য |
| Key Image | ডাবল-ভোট শনাক্তকরণ |
| বেনামিতা | Ring-এর মধ্যে কে স্বাক্ষর করেছে তা অজানা |

### ৪.৬ Post-Quantum (Kyber768)

| বিষয় | বিবরণ |
|---|---|
| অ্যালগরিদম | CRYSTALS-Kyber (ML-KEM) |
| নিরাপত্তা স্তর | NIST Level 3 |
| লাইব্রেরি | CIRCL (Cloudflare) |
| সমস্যা ভিত্তি | Module Learning With Errors (MLWE) |

---

## ৫. শক্তি (Strengths)

### ৫.১ ক্রিপ্টোগ্রাফিক শক্তি
- **সবচেয়ে ব্যাপক ক্রিপ্টো স্ট্যাক**: ৭টি ক্রিপ্টোগ্রাফিক প্রোটোকল একসাথে — কোনো বিদ্যমান সিস্টেমে এত প্রোটোকল একত্রিত হয়নি
- **Formal ZKP**: Σ-Protocol ব্যবহারে গাণিতিকভাবে প্রমাণযোগ্য নিরাপত্তা (Soundness, Completeness, Zero-Knowledge)
- **Homomorphic Tallying**: ভোট ডিক্রিপ্ট না করেই গণনা — কেউ একক ভোট দেখতে পায় না

### ৫.২ বলপ্রয়োগ-প্রতিরোধ (Coercion Resistance)
- **SMDC হলো সবচেয়ে বড় শক্তি** — বেশিরভাগ ই-ভোটিং সিস্টেম এটি দিতে পারে না
- Deniable credentials পদ্ধতি গাণিতিকভাবে indistinguishable
- বাস্তব জীবনে ভোটার বলপ্রয়োগ একটি গুরুতর সমস্যা — CovertVote এর সমাধান দেয়

### ৫.৩ গোপনীয়তা মডেল
- **SA² 2-Server মডেল**: একটি সার্ভার আপোষ হলেও গোপনীয়তা বজায়
- **Ring Signatures**: ভোটারের পরিচয় ১০০ জনের মধ্যে লুকিয়ে থাকে
- **Threshold Decryption**: একক কর্তৃপক্ষ একা ফলাফল দেখতে পারে না

### ৫.৪ পারফরম্যান্স
- **O(n) Linear Complexity**: ISE-Voting-এর O(n × m²)-এর তুলনায় অনেক দক্ষ
- ভোটার সংখ্যা বাড়লেও সময় রৈখিকভাবে বাড়ে, দ্বিঘাতভাবে নয়

### ৫.৫ ভবিষ্যৎ-প্রস্তুততা
- **Post-Quantum Kyber768**: কোয়ান্টাম কম্পিউটারের যুগেও নিরাপদ
- NIST-মানসম্পন্ন PQC অ্যালগরিদম
- Hybrid encryption: বর্তমান + ভবিষ্যৎ উভয় হুমকি মোকাবেলা

### ৫.৬ প্রযুক্তিগত গুণমান
- Go ভাষায় লেখা — উচ্চ পারফরম্যান্স, টাইপ সেফটি, কনকারেন্সি
- ৫১টি টেস্ট, ১০০% পাস রেট
- পরিষ্কার মডিউলার আর্কিটেকচার

---

## ৬. দুর্বলতা (Weaknesses)

### ৬.১ বাস্তবায়ন সীমাবদ্ধতা
- **Threshold Decryption**: শুধু ফ্রেমওয়ার্ক তৈরি হয়েছে, পূর্ণ বাস্তবায়ন নয়
- **SA² Servers**: একই প্রসেসে চলে — বাস্তবে আলাদা মেশিনে হওয়া উচিত
- **Merkle Tree**: Mutex ছাড়া — Concurrent access-এ thread safety ঝুঁকি
- **SQLite Integration**: পূর্ণভাবে সংযুক্ত নয়
- **Docker Containerization**: নেই — production deployment কঠিন

### ৬.২ স্কেলেবিলিটি অপরীক্ষিত
- শুধু ইউনিট টেস্ট আছে, বাস্তব স্কেলে (লক্ষ/কোটি ভোটার) পরীক্ষা হয়নি
- Paillier 2048-bit encryption ধীর — প্রতি ভোটে ~১০০ms+
- Ring size ১০০ ফিক্সড — বড় নির্বাচনে এটি সীমাবদ্ধ

### ৬.৩ নিরাপত্তা ফাঁক
- Formal security proof নেই — শুধু বাস্তবায়ন-ভিত্তিক দাবি
- Side-channel attack analysis হয়নি
- Key management ও secure key storage পরিকল্পনা অসম্পূর্ণ

### ৬.৪ ব্যবহারযোগ্যতা (Usability)
- বায়োমেট্রিক সিস্টেম শুধু SHA-3 হ্যাশিং — বাস্তব fingerprint hardware integration নেই
- সাধারণ ভোটারের জন্য ক্রিপ্টোগ্রাফিক ধারণা জটিল
- ত্রুটি পুনরুদ্ধার (Error Recovery) প্রক্রিয়া সীমিত

### ৬.৫ নির্ভরতা ঝুঁকি
- CIRCL লাইব্রেরি (Kyber) — Cloudflare-এর উপর নির্ভরশীল
- Hyperledger Fabric — জটিল setup, DevOps দক্ষতা প্রয়োজন

---

## ৭. সুবিধা (Pros)

| # | সুবিধা | বিবরণ |
|---|---|---|
| ১ | **সম্পূর্ণ গোপনীয়তা** | কোনো একক সত্তা (সার্ভার, অ্যাডমিন) একক ভোট দেখতে পারে না |
| ২ | **বলপ্রয়োগ প্রতিরোধ** | SMDC-র মাধ্যমে ভোটার চাপমুক্ত ভোট দিতে পারে |
| ৩ | **গাণিতিক যাচাই** | ZKP দিয়ে যেকেউ ভোটের বৈধতা যাচাই করতে পারে |
| ৪ | **অপরিবর্তনীয় রেকর্ড** | Blockchain-এ ভোট সংরক্ষিত — পরিবর্তন অসম্ভব |
| ৫ | **ডাবল-ভোট প্রতিরোধ** | Key Image দিয়ে একই ভোটার দুবার ভোট দিলে ধরা পড়ে |
| ৬ | **ভবিষ্যৎ-নিরাপদ** | Kyber768 কোয়ান্টাম কম্পিউটার থেকে সুরক্ষা দেয় |
| ৭ | **ওপেন সোর্স** | কোড পর্যালোচনাযোগ্য — স্বচ্ছতা |
| ৮ | **দক্ষ** | O(n) complexity — বড় নির্বাচনেও কার্যকর |

---

## ৮. অসুবিধা (Cons)

| # | অসুবিধা | বিবরণ |
|---|---|---|
| ১ | **জটিলতা** | ৭টি ক্রিপ্টোগ্রাফিক প্রোটোকল — বোঝা ও রক্ষণাবেক্ষণ কঠিন |
| ২ | **গণনাগত ব্যয়** | Paillier 2048-bit ধীর, প্রতি ভোটে উল্লেখযোগ্য সময় লাগে |
| ৩ | **অবকাঠামো প্রয়োজন** | Hyperledger Fabric + 2 SA² Server + API Server — জটিল deployment |
| ৪ | **বাস্তব পরীক্ষা নেই** | Lab-স্কেলে পরীক্ষিত, বাস্তব নির্বাচনে ব্যবহার হয়নি |
| ৫ | **Biometric সীমাবদ্ধতা** | সফটওয়্যার-ভিত্তিক — হার্ডওয়্যার সেন্সর integration নেই |
| ৬ | **Internet নির্ভরতা** | অফলাইন ভোটিং সম্ভব নয় |
| ৭ | **Learning Curve** | নির্বাচন কর্মকর্তাদের প্রশিক্ষণ প্রয়োজন |
| ৮ | **আইনি বাধা** | বেশিরভাগ দেশে blockchain ভোটিংয়ের আইনি কাঠামো নেই |

---

## ৯. বিদ্যমান সিস্টেমের তুলনা

### ৯.১ বিস্তারিত তুলনামূলক সারণী

| বৈশিষ্ট্য | **CovertVote** | **ISE-Voting** | **BP-Vot** | **Faruk et al.** | **Voatz** | **Agora** |
|---|---|---|---|---|---|---|
| **Blockchain** | Hyperledger Fabric | Ethereum | Hyperledger Besu | Hyperledger Fabric | Hyperledger | Custom |
| **Encryption** | Paillier HE | — | — | AES-256 | — | Secret Sharing |
| **Homomorphic** | ✅ Additive | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Coercion Resist.** | ✅ SMDC (k=5) | ❌ | ❌ | ❌ | ❌ | ❌ |
| **ZK Proofs** | ✅ Σ-Protocol | ❌ | ❌ | ❌ | ❌ | ❌ |
| **Ring Signatures** | ✅ Linkable (100) | Identity-based | ❌ | ❌ | ❌ | ❌ |
| **Privacy Model** | SA² (2-server) | CSP-based | Differential Privacy | Basic Encryption | Centralized | Secret Sharing |
| **Biometric** | ✅ Fingerprint | ❌ | ❌ | ✅ Dual-factor | ✅ Face+Fingerprint | ❌ |
| **Post-Quantum** | ✅ Kyber768 | আংশিক (symmetric) | ❌ | ❌ | ❌ | ❌ |
| **Complexity** | **O(n)** | O(n × m²) | O(n) | O(n) | অজানা | অজানা |
| **Double-Vote** | Key Image | Smart Contract | Smart Contract | Blockchain | Server-side | Blockchain |
| **Verifiability** | E2E (ZKP) | Contract-based | Limited | Limited | Audit Trail | Partial |
| **Open Source** | ✅ | ❌ | ❌ | ❌ | ❌ | আংশিক |
| **Real Deployment** | ❌ Lab | ❌ Lab | ❌ Lab | ❌ Lab (100 users) | ✅ US Elections | ✅ Sierra Leone |
| **Language** | Go | Solidity | Solidity | Python | Mobile (iOS/Android) | Custom |

### ৯.২ Privacy Model তুলনা

```
গোপনীয়তার মাত্রা (উচ্চ → নিম্ন):

CovertVote (SA² + HE + Ring Sig + SMDC)
    ████████████████████████████████████  ★★★★★

ISE-Voting (CSP-based)
    ████████████████████                  ★★★☆☆

BP-Vot (Differential Privacy)
    ██████████████████████████            ★★★★☆
    (তবে noise যোগে accuracy কমে)

Faruk et al. (Basic Encryption)
    ████████████                          ★★☆☆☆

Voatz (Centralized)
    ████████                              ★★☆☆☆
    (MIT গবেষণায় দুর্বলতা পাওয়া গেছে)

Agora (Secret Sharing)
    ██████████████████                    ★★★☆☆
```

### ৯.৩ Coercion Resistance তুলনা

| সিস্টেম | পদ্ধতি | কার্যকারিতা |
|---|---|---|
| **CovertVote** | SMDC Deniable Credentials (k=5) | ★★★★★ — গাণিতিকভাবে indistinguishable fake credentials |
| **LOKI Vote** | Modified ElGamal + BBS Group Sig | ★★★★☆ — শক্তিশালী কিন্তু implementation জটিল |
| **zkVoting** | zk-SNARKs based | ★★★★☆ — strong কিন্তু gas cost বেশি |
| **JCJ/Civitas** | Fake credentials (তাত্ত্বিক) | ★★★☆☆ — তত্ত্বে শক্তিশালী, বাস্তবায়ন কঠিন |
| **ISE-Voting** | নেই | ★☆☆☆☆ |
| **BP-Vot** | নেই | ★☆☆☆☆ |
| **Voatz** | নেই | ★☆☆☆☆ |

---

## ১০. একাডেমিক পেপারের সাথে তুলনা

### ১০.১ ISE-Voting (Zhang et al., IEEE IoT Journal 2025) এর তুলনায়

| দিক | ISE-Voting | CovertVote | বিজয়ী |
|---|---|---|---|
| **Signature** | Identity-based Ring Sig | Linkable Ring Sig | CovertVote (double-vote detection) |
| **Privacy** | CSP (trusted party) | SA² (no trusted party) | CovertVote |
| **Tallying** | Ballot Cutting Algorithm | Homomorphic Addition | CovertVote (faster) |
| **Complexity** | O(n × m²) | O(n) | **CovertVote** (উল্লেখযোগ্য সুবিধা) |
| **Coercion** | ❌ | ✅ SMDC | **CovertVote** |
| **PQ Security** | আংশিক (symmetric) | ✅ Kyber768 | **CovertVote** |
| **Maturity** | Peer-reviewed (IEEE) | Implementation | ISE-Voting |

**মূল সুবিধা**: CovertVote-এর O(n) complexity ISE-Voting-এর O(n × m²) থেকে **দ্বিঘাতভাবে দ্রুত**। ১ মিলিয়ন ভোটার ও ১০ প্রার্থীতে CovertVote ~১০⁶ অপারেশনে কাজ সারবে, ISE-Voting-এর লাগবে ~১০⁸।

### ১০.২ BP-Vot (IEEE Access 2025) এর তুলনায়

| দিক | BP-Vot | CovertVote | বিজয়ী |
|---|---|---|---|
| **Privacy** | (k,ε)-Differential Privacy | Paillier HE + SA² | CovertVote (no noise) |
| **Accuracy** | ≥98% (noise যোগে কমে) | 100% (exact count) | **CovertVote** |
| **Identity** | SSI (Web3.0 wallet) | Biometric + SMDC | সমান (ভিন্ন পদ্ধতি) |
| **Blockchain** | Hyperledger Besu | Hyperledger Fabric | সমান |
| **Coercion** | ❌ | ✅ SMDC | **CovertVote** |
| **Latency** | 24% improvement claimed | অপরীক্ষিত | BP-Vot (পরিমাপিত) |

**মূল পার্থক্য**: BP-Vot-এর Differential Privacy noise যোগ করে — বড় নির্বাচনে কয়েকটি ভোটের পার্থক্যে ফলাফল ভুল হতে পারে। CovertVote-এর Homomorphic Encryption **১০০% সঠিক ফলাফল** দেয় কোনো noise ছাড়া।

### ১০.৩ Faruk et al. (Cluster Computing 2024) এর তুলনায়

| দিক | Faruk et al. | CovertVote | বিজয়ী |
|---|---|---|---|
| **Biometric** | Fingerprint + Face (dual) | Fingerprint only | Faruk et al. |
| **Real Test** | 100 participants | Unit tests only | Faruk et al. |
| **Crypto Depth** | Basic (AES, SHA) | Advanced (Paillier, ZKP, Ring) | **CovertVote** |
| **Privacy Proof** | ❌ Informal | ZKP-based formal | **CovertVote** |
| **Coercion** | ❌ | ✅ SMDC | **CovertVote** |
| **Post-Quantum** | ❌ | ✅ Kyber768 | **CovertVote** |

---

## ১১. বাস্তবায়িত সিস্টেমের সাথে তুলনা

### ১১.১ Voatz

**কী**: মার্কিন যুক্তরাষ্ট্রের ফেডারেল নির্বাচনে ব্যবহৃত প্রথম ইন্টারনেট ভোটিং অ্যাপ।

| দিক | Voatz | CovertVote |
|---|---|---|
| Real-world use | ✅ US Military overseas voting | ❌ |
| Security audit | ❌ MIT গবেষণায় গুরুতর দুর্বলতা পাওয়া গেছে | টেস্ট-ভিত্তিক যাচাই |
| Open source | ❌ Closed source | ✅ |
| Privacy model | কেন্দ্রীভূত সার্ভার — trust required | SA² decentralized — trustless |
| Coercion | ❌ | ✅ SMDC |
| Crypto depth | Basic | Advanced (৭ প্রোটোকল) |

> **গুরুত্বপূর্ণ**: MIT-এর ২০২০ সালের গবেষণায় Voatz-এ গুরুতর নিরাপত্তা দুর্বলতা পাওয়া গেছে — client-side attack, server compromise, vote manipulation সম্ভব।

### ১১.২ Agora

| দিক | Agora | CovertVote |
|---|---|---|
| Real-world use | ✅ Sierra Leone 2018 পর্যবেক্ষণ | ❌ |
| Privacy | Secret Sharing | SA² + HE + Ring Sig |
| Verifiability | আংশিক | E2E (ZKP) |
| Coercion | ❌ | ✅ SMDC |

### ১১.৩ Follow My Vote

| দিক | Follow My Vote | CovertVote |
|---|---|---|
| Open source | ✅ | ✅ |
| Anonymity | Blind Signatures | Ring Signatures |
| Privacy depth | মাঝারি | উচ্চ (SA² + HE) |
| Coercion | ❌ | ✅ SMDC |

---

## ১২. গবেষণায় অবদান

### ১২.১ মূল অবদানসমূহ

1. **SMDC + Blockchain একত্রীকরণ**: Deniable credentials-কে blockchain ই-ভোটিং-এ প্রথমবারের মতো সম্পূর্ণ বাস্তবায়ন
2. **SA² ভোটিং রূপান্তর**: Apple-এর Federated Learning privacy primitive-কে ভোটিং ডোমেইনে প্রয়োগ
3. **৭-প্রোটোকল স্ট্যাক**: Paillier + Pedersen + ZKP + Ring Sig + SMDC + SA² + Kyber — সবচেয়ে ব্যাপক ক্রিপ্টো স্ট্যাক
4. **O(n) Complexity**: ISE-Voting-এর O(n × m²)-এর তুলনায় উল্লেখযোগ্য উন্নতি
5. **Post-Quantum ই-ভোটিং**: Kyber768 hybrid encryption-সহ সম্পূর্ণ ভোটিং পাইপলাইন

### ১২.২ গবেষণার ফাঁক যেখানে CovertVote অবদান রাখে

| গবেষণার ফাঁক | বিদ্যমান সমাধান | CovertVote-এর সমাধান |
|---|---|---|
| Coercion Resistance + Blockchain | তাত্ত্বিক (JCJ/Civitas) | ✅ SMDC বাস্তবায়ন |
| Exact Tally + Privacy | Differential Privacy (noisy) | ✅ Homomorphic (exact) |
| Post-Quantum + E-voting | খুব কম গবেষণা | ✅ Kyber768 hybrid |
| Multi-protocol Integration | ২-৩টি প্রোটোকল | ✅ ৭টি প্রোটোকল |
| Linear Complexity | O(n×m²) বা worse | ✅ O(n) |

---

## ১৩. সীমাবদ্ধতা ও ভবিষ্যৎ কাজ

### ১৩.১ বর্তমান সীমাবদ্ধতা

| # | সীমাবদ্ধতা | প্রভাব | সমাধানের পরামর্শ |
|---|---|---|---|
| ১ | বাস্তব স্কেলে অপরীক্ষিত | বাস্তব নির্বাচনে কার্যকারিতা অজানা | Large-scale simulation (১০ লক্ষ+ ভোটার) |
| ২ | SA² সার্ভার একই প্রসেসে | নিরাপত্তা মডেল দুর্বল | Docker/Kubernetes-এ আলাদা deployment |
| ৩ | Threshold Decryption অসম্পূর্ণ | একক entity ডিক্রিপ্ট করতে পারে | পূর্ণ t-of-n threshold scheme বাস্তবায়ন |
| ৪ | Formal Security Proof নেই | একাডেমিক গ্রহণযোগ্যতা কম | ProVerif/Tamarin দিয়ে formal verification |
| ৫ | Biometric hardware নেই | বাস্তব ব্যবহার সীমিত | SDK integration (Android/iOS sensor) |
| ৬ | Gas/Transaction cost বিশ্লেষণ নেই | অর্থনৈতিক কার্যকারিতা অজানা | Cost benchmarking |

### ১৩.২ ভবিষ্যৎ গবেষণার দিকনির্দেশনা

1. **Formal Verification**: ProVerif বা Tamarin দিয়ে নিরাপত্তা প্রোটোকলের formal proof
2. **Performance Benchmarking**: বিভিন্ন স্কেলে (১K, ১০K, ১০০K, ১M ভোটার) latency ও throughput পরিমাপ
3. **Lattice-based Ring Signatures**: বর্তমান ring signature-কে post-quantum resistant-এ upgrade
4. **Layer-2 Scaling**: Off-chain computation দিয়ে blockchain bottleneck সমাধান
5. **Accessibility**: প্রতিবন্ধী ভোটারদের জন্য interface design
6. **Regulatory Framework**: আইনি কাঠামো প্রস্তাবনা

---

## ১৪. উপসংহার

**CovertVote** বিদ্যমান blockchain ই-ভোটিং সিস্টেমগুলোর তুলনায় তাত্ত্বিকভাবে সবচেয়ে শক্তিশালী ক্রিপ্টোগ্রাফিক ভিত্তি প্রদান করে। এর SMDC-ভিত্তিক বলপ্রয়োগ-প্রতিরোধ, SA²-ভিত্তিক গোপনীয়তা, এবং Paillier-ভিত্তিক সঠিক গণনা — এই তিনটি একত্রে অন্য কোনো সিস্টেমে পাওয়া যায় না।

তবে, **বাস্তব deployment ও large-scale testing** ছাড়া এই তাত্ত্বিক শ্রেষ্ঠত্ব প্রমাণিত নয়। Voatz ও Agora বাস্তব নির্বাচনে ব্যবহৃত হয়েছে, যেখানে CovertVote এখনো গবেষণাগার পর্যায়ে রয়েছে।

**সারমর্ম**: CovertVote = সবচেয়ে শক্তিশালী তত্ত্ব + সবচেয়ে ব্যাপক ক্রিপ্টো + বাস্তবায়ন ফাঁক

---

## ১৫. তথ্যসূত্র

### একাডেমিক পেপার
1. Zhang et al., "An Improved Secure and Efficient E-Voting Scheme Based on Blockchain Systems," IEEE IoT Journal, 2025
2. Baniata & Caluna, "BP-Vot: Blockchain-Based e-Voting Using Smart Contracts, Differential Privacy and Self-Sovereign Identities," IEEE Access, 2025
3. Faruk et al., "Transforming online voting: a novel system utilizing blockchain and biometric verification," Cluster Computing, 2024

### বিদ্যমান সিস্টেম ও সার্ভে
4. [Blockchain-Based E-Voting Mechanisms: A Survey and a Proposal (MDPI 2024)](https://www.mdpi.com/2673-8732/4/4/21)
5. [Blockchain for securing electronic voting systems: survey (Cluster Computing 2024)](https://link.springer.com/article/10.1007/s10586-024-04709-8)
6. [Articulation of blockchain enabled e-voting systems: SLR (Springer 2025)](https://link.springer.com/article/10.1007/s12083-025-01956-3)

### Coercion Resistance
7. [A Scalable Coercion-Resistant Voting Scheme for Blockchain (ePrint 2023)](https://eprint.iacr.org/2023/1578.pdf)
8. [zkVoting: Zero-knowledge proof based coercion-resistant (ePrint 2024)](https://eprint.iacr.org/2024/1003.pdf)
9. [Efficient, usable and Coercion-Resistant Blockchain-Based E-Voting (ScienceDirect 2025)](https://www.sciencedirect.com/science/article/abs/pii/S2214212625001115)
10. [LOKI Vote: A Blockchain-Based Coercion Resistant E-Voting Protocol](https://www.researchgate.net/publication/347087666_LOKI_Vote_A_Blockchain-Based_Coercion_Resistant_E-Voting_Protocol)

### Homomorphic Encryption ও Privacy
11. [A Timed-Release E-Voting Scheme Based on Paillier HE (IEEE 2024)](https://ieeexplore.ieee.org/iel7/7274860/10712654/10460493.pdf)
12. [Samplable Anonymous Aggregation for Private Federated Data Analysis (ACM CCS 2024)](https://dl.acm.org/doi/10.1145/3658644.3690224)
13. [SA² - Apple Machine Learning Research](https://machinelearning.apple.com/research/samplable-anon-aggregation)

### Post-Quantum
14. [Post-Quantum Secure E-Voting Protocol Using Blockchain and Lattice-Based Cryptography (2025)](https://www.researchgate.net/publication/396831135_Post-Quantum_Secure_E-Voting_Protocol_Using_Blockchain_and_Lattice-Based_Cryptography)
15. [A Quantum-Secure and Blockchain-Integrated E-Voting Framework (arXiv 2025)](https://arxiv.org/abs/2511.16034)
16. [K-Linkable Ring Signatures and Applications in Generalized Voting (ePrint 2025)](https://eprint.iacr.org/2025/243.pdf)

### ZKP ও Ring Signatures
17. [Zero Knowledge Proof on Top of Blockchain for Anonymous E-Voting (Springer 2025)](https://link.springer.com/article/10.1007/s40031-025-01198-0)
18. [Lattice-Based Zero-Knowledge Proofs: Applications to Electronic Voting (J. Cryptology 2024)](https://link.springer.com/article/10.1007/s00145-024-09530-5)
19. [Logarithmic certificate-less linkable ring signature over lattices (ScienceDirect 2025)](https://www.sciencedirect.com/science/article/abs/pii/S0920548925001102)

### Scalability ও Challenges
20. [Secure and Scalable Blockchain Voting: Comparative Framework (arXiv 2025)](https://arxiv.org/abs/2508.05865)
21. [Blockchain-Based E-Voting: Significance and Requirements (Wiley 2024)](https://onlinelibrary.wiley.com/doi/10.1155/2024/5591147)

### Voatz Security Analysis
22. [The Ballot is Busted Before the Blockchain: Security Analysis of Voatz (USENIX Security 2020)](https://www.usenix.org/conference/usenixsecurity20/presentation/specter)

### Biometric Authentication
23. [Comparative E-Voting Security Evaluation: Multi-Modal Biometric (HAL 2024)](https://hal.science/hal-04650059v1/document)
24. [Fingerprint-Authenticated Blockchain E-Voting (ResearchGate 2025)](https://www.researchgate.net/publication/393633419_Fingerprint-Authenticated_Blockchain_E-Voting_A_Secure_Digital_Election_Framework)
