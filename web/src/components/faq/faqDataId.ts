import type { FAQCategoryContent } from './faqData'

/**
 * Pohon konten FAQ bahasa Indonesia. Struktur identik dengan pohon Inggris
 * (tanpa icon — icon dipasang oleh faqData.ts berdasarkan id kategori).
 * Rentang kode ditulis dengan backtick, dirender sebagai gaya monospace.
 */
export const faqIdCategories: FAQCategoryContent[] = [
  // ───────────────────────── Memulai ─────────────────────────
  {
    id: 'getting-started',
    title: 'Memulai',
    items: [
      {
        id: 'what-is-fxos',
        question: 'Apa itu FXOS?',
        blocks: [
          {
            type: 'p',
            text: 'FXOS adalah terminal trading AI open-source yang di-host sendiri. Mode utamanya adalah FXOS Autopilot: agen AI yang membaca papan sinyal Claw402.ai, memverifikasi kandidat dengan Signal Lab dan struktur likuidasi, mengonfirmasi timing dengan candle mentah, dan mengeksekusi di Hyperliquid — semuanya berjalan di mesin Anda sendiri, dengan kunci Anda tidak pernah meninggalkan server Anda.',
          },
          {
            type: 'p',
            text: 'Selain Autopilot, Anda dapat membangun strategi kustom di Strategy Studio, menjalankan beberapa trader AI berdampingan, dan membandingkannya.',
          },
        ],
      },
      {
        id: 'what-do-i-need',
        question: 'Apa saja yang saya butuhkan sebelum meluncurkan Autopilot?',
        blocks: [
          {
            type: 'p',
            text: 'Dua akun yang sudah didanai — peluncuran terpandu di halaman Config akan memandu Anda melalui keduanya:',
          },
          {
            type: 'list',
            items: [
              'Dompet biaya AI: dompet USDC di jaringan Base yang membayar panggilan model AI dan data pasar. Minimum `1 USDC` untuk meluncur.',
              'Akun Hyperliquid dengan otorisasi trading dan setidaknya `12 USDC` tersedia sebagai margin.',
            ],
          },
          {
            type: 'p',
            text: 'Tombol luncur menjalankan preflight sisi server yang memeriksa setiap prasyarat dan mengarahkan Anda ke langkah yang hilang, sehingga Anda tidak dapat memulai bot yang setengah terkonfigurasi.',
          },
        ],
      },
      {
        id: 'which-markets',
        question: 'Pasar apa saja yang bisa diperdagangkan?',
        blocks: [
          {
            type: 'p',
            text: 'Autopilot memperdagangkan perpetual Hyperliquid: kripto utama (BTC, ETH, SOL, …) plus pasar sintetis xyz yang mencakup saham AS, indeks, komoditas, dan FX — satu akun memberi AI alam semesta multi-aset.',
          },
          {
            type: 'p',
            text: 'Trader manual yang dibangun di Strategy Studio juga dapat terhubung ke Binance, Bybit, OKX, Bitget, KuCoin, Gate, Aster, dan Lighter.',
          },
        ],
      },
      {
        id: 'ai-models',
        question: 'Model AI apa yang digunakan? Apakah saya perlu kunci API?',
        blocks: [
          {
            type: 'p',
            text: 'Tidak perlu kunci API. FXOS merutekan inferensi melalui infrastruktur bayar-per-pakai Claw402: dompet biaya AI Anda membayar per panggilan dengan USDC Base, dan terminal mengakses model yang didukung (DeepSeek dan model frontier lainnya) sesuai permintaan.',
          },
          {
            type: 'p',
            text: 'Pengguna tingkat lanjut tetap dapat memasang kunci penyedia sendiri (OpenAI, Claude, Gemini, DeepSeek, Qwen, Grok, Kimi, atau endpoint apa pun yang kompatibel dengan OpenAI) di Config → Models.',
          },
        ],
      },
      {
        id: 'is-it-profitable',
        question: 'Apakah ini akan menghasilkan uang?',
        blocks: [
          {
            type: 'p',
            text: 'Tidak ada yang bisa menjanjikan itu, dan Anda harus curiga pada siapa pun yang menjanjikannya. AI menjalankan proses sistematis, tetapi pasar itu adversial dan kinerja masa lalu tidak pernah menjamin hasil di masa depan.',
          },
          {
            type: 'p',
            text: 'Dasbor sengaja jujur tentang kinerja: memisahkan P/L terealisasi dari yang belum terealisasi, menunjukkan rantai beban biaya (kotor − biaya = bersih), profit factor, dan drawdown maksimum yang dihitung dari saldo awal asli Anda. Perhatikan angka-angka itu, mulai dari kecil, dan hanya trading dengan uang yang sanggup Anda rugikan.',
          },
          {
            type: 'note',
            text: 'Trading mengandung risiko kerugian yang substansial. FXOS adalah perangkat lunak, bukan nasihat investasi.',
          },
        ],
      },
    ],
  },

  // ───────────────────────── Peluncuran & Dompet ─────────────────────────
  {
    id: 'launch-wallets',
    title: 'Peluncuran & Dompet',
    items: [
      {
        id: 'ai-fee-wallet',
        question: 'Apa itu dompet biaya AI?',
        blocks: [
          {
            type: 'p',
            text: 'Dompet EVM khusus di Base yang membayar panggilan model AI dan data pasar berbayar (mikropembayaran x402). Dompet ini sepenuhnya terpisah dari agunan trading Anda — tidak pernah menyentuh Hyperliquid.',
          },
          {
            type: 'list',
            items: [
              'Pengaturan terpandu membuatnya untuk Anda (atau memakai ulang dompet yang sudah ada).',
              'Deposit hanya USDC di jaringan Base ke alamatnya.',
              'Peluncuran membutuhkan setidaknya `1 USDC`; tampilan saldo menyegarkan otomatis setelah deposit.',
              'Satu siklus tipikal berbiaya dari sepersekian sen hingga beberapa sen tergantung modelnya.',
            ],
          },
        ],
      },
      {
        id: 'fee-wallet-private-key',
        question: 'Di mana kunci privat dompet biaya AI disimpan?',
        blocks: [
          {
            type: 'p',
            text: 'Kunci dibuat secara lokal di server Anda, disimpan terenkripsi (AES-256) di database Anda sendiri, dan ditampilkan sekali di layar onboarding. Cadangkan — tidak dapat dipulihkan jika database hilang.',
          },
          {
            type: 'note',
            text: 'Simpan hanya uang biaya di dompet ini. Dompet ini ada untuk membayar panggilan AI, bukan untuk menyimpan tabungan.',
          },
        ],
      },
      {
        id: 'hyperliquid-authorization',
        question: 'Bagaimana cara kerja otorisasi Hyperliquid? Apakah aman?',
        blocks: [
          {
            type: 'p',
            text: 'FXOS menggunakan dompet agen Hyperliquid, sehingga kunci dompet utama Anda tidak pernah dibagikan. Alur koneksi memiliki empat langkah bertanda tangan:',
          },
          {
            type: 'steps',
            items: [
              'Hubungkan dompet EVM Anda (Rabby, MetaMask, OKX, Coinbase Wallet).',
              'Setujui dompet agen FXOS yang baru dibuat — berlaku 180 hari, hanya trading.',
              'Setujui biaya builder (biaya kecil per order yang mendanai platform).',
              'Simpan kunci agen ke server FXOS Anda (disimpan terenkripsi).',
            ],
          },
          {
            type: 'p',
            text: 'Dompet agen dapat membuka dan menutup order, tidak lebih. Tidak dapat menarik dana, dan agunan Anda tetap berada di akun Hyperliquid Anda sendiri setiap saat.',
          },
        ],
      },
      {
        id: 'launch-preflight',
        question: 'Apa yang diperiksa oleh preflight peluncuran?',
        blocks: [
          {
            type: 'p',
            text: 'Sebelum apa pun dibuat atau diubah, server memverifikasi seluruh rantai dengan data langsung:',
          },
          {
            type: 'list',
            items: [
              'Model AI diaktifkan dan memiliki kredensial.',
              'Kunci dompet biaya AI valid dan saldo USDC Base setidaknya `1 USDC` (ditanyakan on-chain).',
              'Akun Hyperliquid diotorisasi (agen + biaya builder) dan dapat dijangkau.',
              'Dana trading: setidaknya `12 USDC` termasuk ekuitas di posisi terbuka.',
            ],
          },
          {
            type: 'p',
            text: 'Setiap pemeriksaan yang gagal menyebutkan perbaikan yang tepat dan menautkan langsung ke pengaturan terpandu. Pemeriksaan yang sama ditegakkan di sisi server setiap kali mulai, sehingga UI tidak dapat dilewati secara tidak sengaja.',
          },
        ],
      },
      {
        id: 'relaunch-behavior',
        question: 'Apa yang terjadi jika saya menekan Launch lagi?',
        blocks: [
          {
            type: 'p',
            text: 'Peluncuran bersifat idempoten. Jika FXOS Autopilot sudah ada, peluncur memperbaruinya dengan konfigurasi strategi saat ini dan memulai ulang — tidak pernah membuat duplikat. Restart bisa memakan waktu hingga satu menit jika bot sedang di tengah siklus; UI menunggunya.',
          },
        ],
      },
      {
        id: 'deposit-not-showing',
        question: 'Saya deposit USDC tetapi saldo masih nol.',
        blocks: [
          {
            type: 'list',
            items: [
              'Dompet biaya AI: pastikan Anda mengirim USDC di jaringan Base ke alamat persis yang ditampilkan. Saldo di-cache sekitar 30 detik, dan panel pengaturan memeriksa ulang otomatis setiap beberapa detik.',
              'Hyperliquid: deposit masuk ke akun Hyperliquid Anda sendiri; langkah saldo memantau status akun langsung. Gunakan Refresh di panel terpandu jika ragu.',
              'Jika RPC on-chain untuk sementara tidak dapat dijangkau, panel menandai saldo sebagai tidak diketahui, bukan nol — coba lagi dalam satu menit.',
            ],
          },
        ],
      },
    ],
  },

  // ───────────────────────── Trading & Eksekusi ─────────────────────────
  {
    id: 'trading',
    title: 'Trading & Eksekusi',
    items: [
      {
        id: 'autopilot-pipeline',
        question: 'Bagaimana cara kerja strategi Autopilot, langkah demi langkah?',
        blocks: [
          {
            type: 'p',
            text: 'Setiap siklus menjalankan corong empat tahap yang sama — setiap tahap dapat menolak kandidat, dan hanya setup yang lolos keempat tahap yang diperdagangkan:',
          },
          {
            type: 'steps',
            items: [
              'Bangun alam semesta — tarik peringkat Claw402.ai langsung dan ambil kandidat teratas (default 10) di seluruh kripto utama dan pasar sintetis xyz (saham AS, indeks, komoditas, FX), masing-masing dengan bias arah dan z-score sinyal.',
              'Verifikasi setiap kandidat — ambil sinyal dalam Signal Lab plus struktur biaya/likuidasi di sekitar harga: kluster likuidasi dan basis biaya menunjukkan apakah pergerakan memiliki bahan bakar di depannya atau tembok yang melawannya.',
              'Konfirmasi timing — baca candle OHLCV 15 menit mentah (30 bar) untuk memastikan entri mengikuti struktur, bukan mengejar pergerakan yang sudah ekstrem.',
              'Putuskan dan ukur — hanya setup yang melewati ambang keyakinan (default `78/100`) dengan rasio risiko/imbalan sekitar `3:1` yang mendapat posisi di 10x; penutupan selalu dieksekusi sebelum pembukaan, dan buku long serta short dipertimbangkan setiap siklus.',
            ],
          },
          {
            type: 'p',
            text: 'Lapisan kelima berada di luar AI sepenuhnya: kontrol risiko keras (batas posisi, batas leverage, batas margin, throttle trading) menolak keputusan apa pun yang melanggarnya, sepercaya apa pun modelnya.',
          },
        ],
      },
      {
        id: 'data-sources',
        question: 'Data apa yang digunakan, dan bagian mana yang berbayar?',
        blocks: [
          {
            type: 'list',
            items: [
              'Data sinyal Claw402.ai — papan peringkat, sinyal dalam Signal Lab per simbol, dan net-flow pasar. Ini endpoint berbayar, ditagih per panggilan dalam USDC dari dompet biaya AI Anda (mikropembayaran x402).',
              'Peta panas biaya/likuidasi — struktur biaya posisi agregat dan kluster likuidasi untuk setiap pasar.',
              'Data pasar Hyperliquid — candle OHLCV mentah dan buku order L2 langsung (data publik gratis).',
              'Akun Anda melalui dompet agen — ekuitas, margin tersedia, posisi terbuka dengan PnL.',
              'Riwayatnya sendiri — perdagangan tertutup memberi makan win rate, profit factor, dan drawdown kembali ke prompt berikutnya, sehingga AI mengetahui performa terkininya.',
            ],
          },
          {
            type: 'note',
            text: 'Dasbor memantau endpoint Claw402 berbayar dengan lambat (setiap beberapa menit) untuk menghemat dompet biaya Anda — panel data pasar tetap hidup dari umpan gratis.',
          },
        ],
      },
      {
        id: 'decision-cycle',
        question: 'Seberapa sering AI membuat keputusan?',
        blocks: [
          {
            type: 'p',
            text: 'Autopilot menjalankan siklus pemindaian setiap 5–15 menit tergantung cara Anda meluncurkannya (dapat dikonfigurasi per trader, minimum 3 menit). Siklus pertama dimulai tepat setelah peluncuran; satu siklus biasanya memakan 30–60 detik karena AI membaca konteks pasar penuh sebelum memutuskan.',
          },
        ],
      },
      {
        id: 'what-ai-sees',
        question: 'Informasi apa yang AI lihat setiap siklus?',
        blocks: [
          {
            type: 'list',
            items: [
              'Akun Anda: ekuitas, margin tersedia, posisi terbuka dengan PnL.',
              'Papan peringkat Claw402: alam semesta kandidat dengan bias arah.',
              'Sinyal dalam Signal Lab dan struktur biaya/likuidasi per kandidat.',
              'Candle OHLCV mentah untuk konfirmasi timing.',
              'Rekam jejaknya sendiri: win rate, profit factor, drawdown, perdagangan terbaru.',
            ],
          },
          {
            type: 'p',
            text: 'Setiap siklus disimpan sebagai catatan keputusan — Execution Log di dasbor menampilkan rantai penalaran, tindakan, dan order yang diblokir.',
          },
        ],
      },
      {
        id: 'leverage-and-risk',
        question: 'Leverage dan kontrol risiko apa yang digunakan?',
        blocks: [
          {
            type: 'p',
            text: 'Autopilot default ke margin silang 10x. Kontrol risiko keras berjalan di luar AI dan tidak dapat ditimpa olehnya:',
          },
          {
            type: 'list',
            items: [
              'Batas jumlah posisi dari konfigurasi strategi — pembukaan baru ditolak saat mencapai batas.',
              'Batas leverage per kelas aset (BTC/ETH vs altcoin).',
              'Throttle trading memblokir churn, misalnya menutup posisi yang nyaris tidak bergerak beberapa menit setelah dibuka.',
              'Safe mode (di bawah) melindungi buku saat AI itu sendiri gagal.',
            ],
          },
        ],
      },
      {
        id: 'safe-mode',
        question: 'Apa itu safe mode?',
        blocks: [
          {
            type: 'p',
            text: 'Jika AI gagal 3 siklus berturut-turut (pemadaman penyedia, dompet biaya kosong, respons buruk), trader masuk ke safe mode: tidak ada posisi baru yang dibuka, posisi yang ada tetap terlindungi, dan loop terus mencoba ulang. Safe mode keluar otomatis pada panggilan AI sukses berikutnya.',
          },
          {
            type: 'p',
            text: 'Safe mode ditampilkan sebagai banner di dasbor bersama alasannya, sehingga tidak pernah terjadi secara diam-diam.',
          },
        ],
      },
      {
        id: 'fee-wallet-empty-mid-run',
        question: 'Apa yang terjadi jika dompet biaya AI habis di tengah jalan?',
        blocks: [
          {
            type: 'p',
            text: 'Panggilan AI mulai gagal dengan status "out of funds" yang jelas. Dasbor menampilkan banner merah persisten dengan saldo dompet, dan setelah tiga siklus gagal bot masuk ke safe mode. Isi ulang USDC Base ke dompet biaya dan trader pulih sendiri — tanpa perlu restart.',
          },
        ],
      },
      {
        id: 'trading-fees',
        question: 'Biaya apa yang saya bayarkan?',
        blocks: [
          {
            type: 'list',
            items: [
              'Biaya trading Hyperliquid pada setiap order, plus biaya builder yang disetujui.',
              'Biaya AI/data dibayar per panggilan dari dompet biaya (sen per siklus).',
            ],
          },
          {
            type: 'p',
            text: 'Biaya adalah pembunuh diam-diam strategi frekuensi tinggi. Strip statistik dasbor menampilkan rantai penuh — P/L terealisasi kotor, dikurangi biaya, sama dengan bersih — sehingga Anda dapat segera melihat apakah biaya memakan keunggulan.',
          },
        ],
      },
      {
        id: 'stop-and-manual',
        question: 'Bagaimana cara menghentikan bot atau menutup posisi secara manual?',
        blocks: [
          {
            type: 'list',
            items: [
              'Berhenti: gunakan tombol Stop di daftar trader halaman Config. Berhenti menghentikan loop keputusan; posisi terbuka tetap terbuka dan menjadi tanggung jawab Anda.',
              'Tutup manual: tutup posisi apa pun dari panel posisi dasbor — penutupan manual tersinkron kembali ke riwayat posisi.',
              'Darurat: Anda selalu dapat mengelola posisi langsung di Hyperliquid; FXOS tidak pernah mengunci Anda dari akun Anda sendiri.',
            ],
          },
        ],
      },
    ],
  },

  // ───────────────────────── Dasbor & Metrik ─────────────────────────
  {
    id: 'dashboard',
    title: 'Dasbor & Metrik',
    items: [
      {
        id: 'metrics-meaning',
        question: 'Apa arti metrik di header secara tepat?',
        blocks: [
          {
            type: 'list',
            items: [
              'Ekuitas — nilai akun langsung termasuk PnL belum terealisasi.',
              'Total P/L (termasuk belum terealisasi) — ekuitas versus saldo awal Anda; bergerak dengan posisi terbuka.',
              'P/L terealisasi (perdagangan tertutup) — hasil bersih hanya dari perdagangan selesai; dari sinilah win rate, profit factor, dan sharpe dihitung.',
              'Profit factor — kemenangan kotor ÷ kerugian kotor pada perdagangan tertutup; di atas 1.0 berarti buku tertutup positif bersih.',
              'Drawdown maksimum — penurunan puncak-ke-lembah terburuk dari kurva ekuitas terealisasi, diukur terhadap saldo awal asli Anda.',
            ],
          },
        ],
      },
      {
        id: 'pl-contradiction',
        question: 'Mengapa Total P/L positif sementara Realized P/L negatif?',
        blocks: [
          {
            type: 'p',
            text: 'Keduanya mengukur hal yang berbeda. Realized P/L hanya menghitung perdagangan tertutup; Total P/L juga mencakup keuntungan belum terealisasi dari posisi yang masih terbuka. Bot bisa rugi pada perdagangan tertutup sementara buku terbukanya membawa cukup laba belum terealisasi untuk membuat total P/L hijau — dan sebaliknya. Periksa strip kotor/biaya/bersih untuk melihat berapa banyak hasil terealisasi yang merupakan beban biaya.',
          },
        ],
      },
      {
        id: 'execution-log',
        question: 'Di mana saya bisa melihat mengapa AI melakukan (atau menolak) sesuatu?',
        blocks: [
          {
            type: 'p',
            text: 'Panel Execution Log mencantumkan setiap siklus dengan tindakan yang diambil, durasi panggilan AI, dan order yang diblokir dengan pengaman spesifik yang memicunya (throttle, batas posisi, kontrol risiko). Rantai penalaran penuh disimpan dengan setiap catatan keputusan.',
          },
        ],
      },
    ],
  },

  // ───────────────────────── Keamanan ─────────────────────────
  {
    id: 'security',
    title: 'Keamanan',
    items: [
      {
        id: 'key-storage',
        question: 'Bagaimana kunci saya disimpan?',
        blocks: [
          {
            type: 'list',
            items: [
              'Semua rahasia (kunci agen, kunci dompet biaya, kunci API bursa) dienkripsi AES-256 saat disimpan di database Anda sendiri.',
              'Enkripsi transport RSA opsional melindungi rahasia dalam perjalanan antara browser dan server.',
              'FXOS di-host sendiri: tidak ada yang dikirim ke server pihak ketiga mana pun. Kode sumber terbuka dan dapat diaudit.',
            ],
          },
        ],
      },
      {
        id: 'can-fxos-steal-funds',
        question: 'Bisakah FXOS menarik atau mencuri dana saya?',
        blocks: [
          {
            type: 'p',
            text: 'Tidak. Di Hyperliquid, FXOS hanya memegang dompet agen, yang menurut desain protokol dapat trading tetapi tidak dapat menarik. Agunan Anda tetap di akun Anda sendiri di bawah kendali dompet utama Anda.',
          },
          {
            type: 'note',
            text: 'Jika Anda menghubungkan bursa terpusat (CEX), buat kunci API dengan izin trading saja — nonaktifkan penarikan dan atur whitelist IP.',
          },
        ],
      },
      {
        id: 'registration-model',
        question: 'Mengapa orang lain tidak bisa mendaftar di instance saya?',
        blocks: [
          {
            type: 'p',
            text: 'Secara desain, sebuah instance adalah single-operator: akun pertama yang terdaftar menjadi operator dan pendaftaran ditutup ("System already initialized"). Ini mencegah orang asing membuat akun di deployment yang terekspos. Jalankan satu instance per operator.',
          },
        ],
      },
    ],
  },

  // ───────────────────────── Self-hosting & Pemecahan Masalah ─────────────────────────
  {
    id: 'self-hosting',
    title: 'Self-Hosting & Pemecahan Masalah',
    items: [
      {
        id: 'how-to-install',
        question: 'Bagaimana cara menginstal FXOS?',
        blocks: [
          {
            type: 'p',
            text: 'Satu baris di Linux/macOS (menginstal dan memulai semuanya melalui Docker):',
          },
          {
            type: 'list',
            items: [
              'Skrip: `curl -fsSL https://raw.githubusercontent.com/onecany/fxos/main/scripts/install.sh | bash`',
              'Docker: unduh `docker-compose.prod.yml` dan jalankan `docker compose -f docker-compose.prod.yml up -d`',
              'Windows: instal Docker Desktop, lalu gunakan rute Docker di atas.',
              'Dari sumber: Go 1.26+, Node 20+, TA-Lib (`brew install ta-lib` / `apt-get install libta-lib0-dev`), lalu `go run .` dan `npm --prefix web run dev`.',
            ],
          },
          {
            type: 'p',
            text: 'Lalu buka `http://127.0.0.1:3000` — UI web di port 3000, API di 8080.',
          },
        ],
      },
      {
        id: 'how-to-update',
        question: 'Bagaimana cara memperbarui?',
        blocks: [
          {
            type: 'p',
            text: 'Jalankan ulang skrip instalasi, atau dengan Docker: `docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml up -d`. Database dan kunci Anda berada di direktori `data/` yang di-mount dan bertahan dari pembaruan. Trader yang berjalan dimulai ulang otomatis setelah backend kembali.',
          },
        ],
      },
      {
        id: 'launch-blocked',
        question: 'Peluncuran diblokir oleh pemeriksaan yang gagal — bagaimana?',
        blocks: [
          {
            type: 'p',
            text: 'Baca pesannya: setiap kegagalan preflight menyebutkan perbaikannya dan mengarahkan Anda ke langkah pengaturan yang tepat — danai dompet AI, selesaikan otorisasi Hyperliquid, atau deposit USDC trading. Saldo diperiksa ulang secara langsung, jadi begitu Anda memperbaikinya, peluncuran akan berhasil.',
          },
        ],
      },
      {
        id: 'exchange-unreachable',
        question: 'Akun bursa menampilkan "invalid credentials" atau "unavailable".',
        blocks: [
          {
            type: 'list',
            items: [
              'Kredensial tidak valid: otorisasi agen kedaluwarsa (180 hari) atau kunci tersimpan basi — hubungkan ulang dompet Hyperliquid; alurnya menawarkan perpanjangan sekali klik.',
              'Tidak tersedia: API bursa tidak merespons; status akun di-cache 30 detik, jadi tunggu dan segarkan.',
              'Kunci CEX: verifikasi izin trading, whitelist IP, dan akses futures/perp diaktifkan.',
            ],
          },
        ],
      },
      {
        id: 'where-are-logs',
        question: 'Di mana lognya?',
        blocks: [
          {
            type: 'list',
            items: [
              'Backend: `docker logs fxos-trading` (atau terminal yang menjalankan `go run .`).',
              'Penalaran dan kesalahan AI per siklus: Execution Log di dasbor.',
              'Masalah build/runtime frontend: konsol devtools browser.',
            ],
          },
        ],
      },
      {
        id: 'port-conflicts',
        question: 'Port 3000 atau 8080 sudah digunakan.',
        blocks: [
          {
            type: 'p',
            text: 'Hentikan layanan yang bentrok atau petakan ulang port yang dipublikasikan di file compose Anda (misalnya `"3100:80"` untuk frontend, `"8180:8080"` untuk API), lalu mulai ulang kontainer.',
          },
        ],
      },
    ],
  },

  // ───────────────────────── Berkontribusi ─────────────────────────
  {
    id: 'contributing',
    title: 'Berkontribusi',
    items: [
      {
        id: 'how-to-contribute',
        question: 'Bagaimana cara berkontribusi kode?',
        blocks: [
          {
            type: 'links',
            links: [
              { label: 'Roadmap', href: 'https://github.com/orgs/onecany/projects/3' },
              { label: 'Task Dashboard', href: 'https://github.com/orgs/onecany/projects/5' },
              { label: 'CONTRIBUTING.md', href: 'https://github.com/onecany/fxos/blob/dev/docs/CONTRIBUTING.md' },
            ],
          },
          {
            type: 'steps',
            items: [
              'Pilih tugas dari papan di atas (filter dengan good first issue / help wanted) dan komentar "assign me".',
              'Fork repo dan buat cabang dari `dev`: `git checkout -b feat/your-topic`.',
              'Ikuti Conventional Commits; jalankan `npm --prefix web run lint && npm --prefix web run build` sebelum push.',
              'Buka PR ke `onecany/fxos:dev`, rujuk issue (`Closes #123`), dan lampirkan screenshot untuk perubahan UI.',
            ],
          },
        ],
      },
      {
        id: 'bounty-program',
        question: 'Apakah ada program bounty?',
        blocks: [
          {
            type: 'p',
            text: 'Ya — issue terpilih membawa bounty uang tunai, plus lencana, peninjauan prioritas, dan akses beta bagi kontributor rutin.',
          },
          {
            type: 'links',
            links: [
              { label: 'Issues dengan label bounty', href: 'https://github.com/onecany/fxos/labels/bounty' },
              { label: 'Templat klaim bounty', href: 'https://github.com/onecany/fxos/blob/dev/.github/ISSUE_TEMPLATE/bounty_claim.md' },
            ],
          },
        ],
      },
      {
        id: 'report-bugs',
        question: 'Bagaimana cara melaporkan bug?',
        blocks: [
          {
            type: 'p',
            text: 'Buka issue GitHub dengan templat: apa yang Anda lakukan, apa yang terjadi, log backend (`docker logs fxos-trading`), dan screenshot. Untuk dugaan masalah keamanan, ikuti catatan responsible-disclosure di SECURITY.md alih-alih issue publik.',
          },
          {
            type: 'links',
            links: [
              { label: 'Issue baru', href: 'https://github.com/onecany/fxos/issues/new/choose' },
              { label: 'SECURITY.md', href: 'https://github.com/onecany/fxos/blob/dev/docs/SECURITY.md' },
            ],
          },
        ],
      },
    ],
  },
]
