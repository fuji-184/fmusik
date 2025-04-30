<script>
	import { connect_ws, send_msg } from "$lib/ws.svelte.js";
	import { onMount } from "svelte";

	let msg = "";
	let messages = $state([]);
	let connected = false;
	let canPlayAudio = false;
	let singlePlayerMode = $state(false);
	
	// Web Audio API Context
	let audioContext;
	
	// Audio buffers cache
	let audioBuffers = $state({
		drum: null,
		piano: null
	});
	
	// Status loading
	let loading = $state({
		drum: false,
		piano: false
	});
	
	// Daftar sumber audio yang sedang diputar
	let activeSources = $state([]);

	// Inisialisasi Audio Context setelah interaksi pengguna
	function initAudio() {
		if (!audioContext) {
			audioContext = new (window.AudioContext || window.webkitAudioContext)();
			canPlayAudio = true;
			
			// Pre-load audio files
			loadAudioFile("drum", "http://127.0.0.1:3000/files/audio1.mp3");
			loadAudioFile("piano", "http://127.0.0.1:3000/files/audio2.mp3");
			
			// Setup interval untuk membersihkan sumber yang sudah selesai
			setupCleanupInterval();
		}
	}
	
	// Fungsi untuk load dan decode audio file
	async function loadAudioFile(name, url) {
		if (audioBuffers[name]) {
			console.log(`${name} already loaded`);
			return;
		}
		
		if (loading[name]) {
			console.log(`${name} is already loading`);
			return new Promise(resolve => {
				// Poll until loading is complete
				const checkInterval = setInterval(() => {
					if (audioBuffers[name]) {
						clearInterval(checkInterval);
						resolve();
					}
				}, 100);
			});
		}
		
		console.log(`Start loading ${name} audio`);
		loading[name] = true;
		
		try {
			console.time(`load-${name}`);
			
			// Use local audio context for loading
			const ctx = audioContext;
			
			// Fetch the file
			const response = await fetch(url);
			console.log(`${name} fetched, decoding...`);
			const arrayBuffer = await response.arrayBuffer();
			
			// Decode the audio data
			const audioBuffer = await ctx.decodeAudioData(arrayBuffer);
			console.log(`${name} decoded`);
			
			// Store the buffer
			audioBuffers[name] = audioBuffer;
			console.timeEnd(`load-${name}`);
		} catch (error) {
			console.error(`Error loading audio ${name}:`, error);
		} finally {
			loading[name] = false;
		}
	}

	// Fungsi untuk memutar audio dari buffer tanpa mengganggu audio lain
	function playAudioBuffer(name) {
		if (!audioContext || !audioBuffers[name]) {
			console.warn(`Cannot play ${name} - context or buffer missing`);
			return;
		}
		
		console.time(`play-${name}`);
		
		// Buat source node baru dari buffer
		const source = audioContext.createBufferSource();
		source.buffer = audioBuffers[name];
		
		// Hubungkan ke output
		source.connect(audioContext.destination);
		
		// Tambahkan ke daftar sumber aktif dengan timestamp dan durasi
		const duration = audioBuffers[name].duration * 1000; // konversi ke ms
		const sourceId = Date.now() + Math.random();
		const sourceInfo = {
			source,
			name,
			id: sourceId,
			startTime: Date.now(),
			endTime: Date.now() + duration
		};
		
		// Mulai pemutaran dengan 0 delay
		try {
			source.start(0); // Specify 0 delay
			console.log(`${name} playback started at ${new Date().toISOString()}`);
			
			// Add to active sources after successful start
			activeSources = [...activeSources, sourceInfo];
			
			// Hapus dari daftar setelah selesai
			source.onended = () => {
				console.log(`${name} playback ended naturally`);
				activeSources = activeSources.filter(s => s.id !== sourceId);
			};
			
			// Backup: Jika onended tidak dipanggil, hapus setelah durasi + margin
			setTimeout(() => {
				if (activeSources.some(s => s.id === sourceId)) {
					console.log(`${name} cleanup by timeout`);
					activeSources = activeSources.filter(s => s.id !== sourceId);
				}
			}, duration + 100); // tambahkan sedikit margin
			
			console.timeEnd(`play-${name}`);
		} catch (error) {
			console.error(`Error playing ${name}:`, error);
		}
	}

	// Handler untuk pesan audio yang diterima
	function handleAudioMessage(data) {
		if (!canPlayAudio) return;
		
		// Resume context jika suspended (kebijakan browser)
		if (audioContext && audioContext.state === "suspended") {
			audioContext.resume();
		}
		
		// Debug information
		console.log(`Received audio command: ${data} at ${new Date().toISOString()}`);
		
		if (data === "drum") {
			if (audioBuffers.drum) {
				console.log("Playing drum from buffer");
				playAudioBuffer("drum");
			} else if (!loading.drum) {
				console.log("Loading drum file first");
				// Tampilkan status loading
				loading.drum = true;
				loadAudioFile("drum", "http://127.0.0.1:3000/files/audio.mp3")
					.then(() => {
						console.log("Drum loaded, now playing");
						playAudioBuffer("drum");
					});
			} else {
				console.log("Drum is currently loading");
			}
		} else if (data === "piano") {
			if (audioBuffers.piano) {
				console.log("Playing piano from buffer");
				playAudioBuffer("piano");
			} else if (!loading.piano) {
				console.log("Loading piano file first");
				// Tampilkan status loading
				loading.piano = true;
				loadAudioFile("piano", "http://127.0.0.1:3000/files/audio2.wav")
					.then(() => {
						console.log("Piano loaded, now playing");
						playAudioBuffer("piano");
					});
			} else {
				console.log("Piano is currently loading");
			}
		}
	}

	function connect() {
		if (!connected) {
			connect_ws("grup1", (data) => {
				messages = [...messages, data];
				handleAudioMessage(data);
			});
			connected = true;
		}
	}

	function chat() {
		send_msg(msg);
		msg = "";
	}

	function drum() {
		if (singlePlayerMode) {
			// Mode single player: langsung putar audio
			handleAudioMessage("drum");
		} else {
			// Mode multiplayer: kirim pesan lewat WebSocket
			send_msg("drum");
		}
	}

	function piano() {
		if (singlePlayerMode) {
			// Mode single player: langsung putar audio
			handleAudioMessage("piano");
		} else {
			// Mode multiplayer: kirim pesan lewat WebSocket
			send_msg("piano");
		}
	}

	function stopAllSounds() {
		if (!audioContext) return;
		
		// Hentikan semua sumber audio yang sedang bermain
		activeSources.forEach(sourceInfo => {
			try {
				sourceInfo.source.stop();
			} catch (e) {
				console.log("Source already stopped");
			}
		});
		
		// Bersihkan daftar sumber
		activeSources = [];
	}
	
	// Fungsi untuk membersihkan sumber audio yang sudah selesai
	function cleanupFinishedSources() {
		const now = Date.now();
		const oldSourcesCount = activeSources.length;
		
		// Filter out sources that should be finished based on their endTime
		activeSources = activeSources.filter(source => {
			return now < source.endTime;
		});
		
		// Log jika ada yang dibersihkan
		if (oldSourcesCount !== activeSources.length) {
			console.log(`Cleaned up ${oldSourcesCount - activeSources.length} finished audio sources`);
		}
	}
	
	// Setup interval untuk membersihkan sumber yang selesai secara berkala
	let cleanupInterval;
	
	function setupCleanupInterval() {
		if (!cleanupInterval) {
			cleanupInterval = setInterval(cleanupFinishedSources, 1000);
		}
	}
	
	// Toggle mode (single player / multiplayer)
	async function toggleMode() {
		singlePlayerMode = !singlePlayerMode;
		
		// Jika beralih ke mode single player, pastikan audio sudah diinisialisasi
		if (singlePlayerMode && !audioContext) {
			await initAudio();
		}
	}
	
	// Clean up saat komponen dihancurkan
	onMount(() => {
		return () => {
			stopAllSounds();
			if (cleanupInterval) {
				clearInterval(cleanupInterval);
			}
			if (audioContext) {
				audioContext.close();
			}
		};
	});
</script>

<div class="p-6 max-w-lg mx-auto bg-white rounded-xl shadow-md">
	<h2 class="text-xl font-bold mb-4">Audio Player</h2>
	
	<div class="bg-gray-100 p-4 rounded-lg mb-4">
		<div class="flex items-center mb-2">
			<span class="mr-2">Mode:</span>
			<button 
				class={`px-3 py-1 rounded-l-md ${singlePlayerMode ? 'bg-gray-300' : 'bg-blue-500 text-white'}`}
				on:click={() => {if (singlePlayerMode) toggleMode()}}
			>
				Multiplayer
			</button>
			<button 
				class={`px-3 py-1 rounded-r-md ${singlePlayerMode ? 'bg-blue-500 text-white' : 'bg-gray-300'}`}
				on:click={() => {if (!singlePlayerMode) toggleMode()}}
			>
				Single Player
			</button>
		</div>
		
		{#if !singlePlayerMode}
			<div class="flex gap-2 mb-4">
				<button 
					class="bg-blue-600 px-4 py-2 text-white rounded disabled:bg-gray-400" 
					on:click={connect}
					disabled={connected}
				>
					{connected ? 'Connected' : 'Connect to WebSocket'}
				</button>
			</div>
		{/if}
		
		<div class="flex gap-2 mb-4">
			<button 
				class="bg-blue-600 px-4 py-2 text-white rounded disabled:bg-gray-400" 
				on:click={initAudio}
				disabled={canPlayAudio}
			>
				{canPlayAudio ? 'Audio Initialized' : 'Initialize Audio'}
			</button>
			<button class="bg-red-600 px-4 py-2 text-white rounded" on:click={stopAllSounds}>
				Stop All Sounds
			</button>
		</div>
	</div>
	
	<div class="flex gap-4 mb-8">
		<button 
			class="bg-red-500 px-6 py-4 text-white rounded-lg flex-1 text-lg font-bold shadow-md hover:bg-red-600 transition-colors" 
			on:click={drum}
			disabled={!canPlayAudio && singlePlayerMode || !connected && !singlePlayerMode}
		>
			Drum
		</button>
		<button 
			class="bg-green-500 px-6 py-4 text-white rounded-lg flex-1 text-lg font-bold shadow-md hover:bg-green-600 transition-colors" 
			on:click={piano}
			disabled={!canPlayAudio && singlePlayerMode || !connected && !singlePlayerMode}
		>
			Piano
		</button>
	</div>
	
	<div class="bg-gray-100 p-4 rounded-lg">
		<h3 class="font-bold mb-2">Status</h3>
		<div class="grid grid-cols-2 gap-2 text-sm">
			<p>Mode: <span class="font-semibold">{singlePlayerMode ? 'Single Player' : 'Multiplayer'}</span></p>
			<p>Audio Context: <span class="font-semibold">{audioContext ? audioContext.state : "Not Initialized"}</span></p>
			<p>Connected: <span class="font-semibold">{connected ? "Yes" : "No"}</span></p>
			<p>Audio Enabled: <span class="font-semibold">{canPlayAudio ? "Yes" : "No"}</span></p>
			<p>Active Audio Sources: <span class="font-semibold">{activeSources.length}</span></p>
			<p>Loading: <span class="font-semibold">{Object.entries(loading).filter(([_, isLoading]) => isLoading).map(([name]) => name).join(', ') || 'None'}</span></p>
		</div>
		
		<p class="mt-2">Currently Playing: <span class="font-semibold">{activeSources.map(s => s.name).join(', ') || 'None'}</span></p>
		<p class="mt-1">Cached Buffers: <span class="font-semibold">{Object.entries(audioBuffers).filter(([_, buffer]) => buffer !== null).map(([name]) => name).join(', ') || 'None'}</span></p>
		
		<div class="mt-2 flex gap-2">
			<button class="bg-blue-500 text-white px-3 py-1 text-sm rounded" on:click={cleanupFinishedSources}>
				Force Cleanup Sources
			</button>
			
			<button class="bg-orange-500 text-white px-3 py-1 text-sm rounded" on:click={() => console.clear()}>
				Clear Console
			</button>
		</div>
	</div>
	
	{#if !singlePlayerMode}
		<div class="mt-4 bg-gray-100 p-4 rounded-lg">
			<h3 class="font-bold mb-2">Messages:</h3>
			<ul class="list-disc pl-5 max-h-40 overflow-y-auto">
				{#each messages as message}
					<li>{message}</li>
				{/each}
			</ul>
		</div>
	{/if}
</div>