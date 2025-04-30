<script>
    import { onMount } from 'svelte';
    import { connect_ws, send_msg, get_msgs } from '$lib/ws.svelte.js'; // Import WebSocket utilities

    // Note frequencies for each key (A0 to C8)
    const createNoteTable = () => {
        const noteFreq = new Map();
        const A4 = 440;
        const SEMITONE = Math.pow(2, 1/12);
        const OCTAVE = Math.pow(2, 1);
        
        // Calculate frequencies for all notes
        for (let octave = 0; octave < 8; octave++) {
            let noteNames = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B'];
            noteNames.forEach((note, i) => {
                const distanceFromA4 = (octave - 4) * 12 + i - 9;
                const freq = A4 * Math.pow(SEMITONE, distanceFromA4);
                noteFreq.set(`${note}${octave}`, freq);
            });
        }
        return noteFreq;
    };

    const noteFrequencies = createNoteTable();

    // Keyboard mapping (2 octaves for example)
    const keyboardMap = {
        'z': 'C4', 's': 'C#4', 'x': 'D4', 'd': 'D#4', 'c': 'E4', 'v': 'F4',
        'g': 'F#4', 'b': 'G4', 'h': 'G#4', 'n': 'A4', 'j': 'A#4', 'm': 'B4',
        'q': 'C5', '2': 'C#5', 'w': 'D5', '3': 'D#5', 'e': 'E5', 'r': 'F5',
        '5': 'F#5', 't': 'G5', '6': 'G#5', 'y': 'A5', '7': 'A#5', 'u': 'B5'
    };

    let audioContext;
    let masterGainNode;
    let activeNotes = new Map();
    let cleanupTimers = new Map();
    
    // WebSocket related variables
    let roomName = "piano_room";
    let connected = false;
    let receivedNotes = [];
    
    onMount(() => {
        initAudio();
        window.addEventListener('keydown', handleKeyDown);
        window.addEventListener('keyup', handleKeyUp);
        
        // Connect to WebSocket
        connect_ws(roomName, handleWebSocketMessage);
        connected = true;
        
        return () => {
            window.removeEventListener('keydown', handleKeyDown);
            window.removeEventListener('keyup', handleKeyUp);
            stopAllNotes();
            cleanupAllNotes();
            if (audioContext) {
                audioContext.close();
            }
        };
    });

    // Handle incoming WebSocket messages
    function handleWebSocketMessage(data) {
        try {
            const message = JSON.parse(data);
            
            if (message.type === 'noteOn') {
                startNote(message.note, message.velocity || 0.7);
                receivedNotes = [...receivedNotes, { note: message.note, timestamp: new Date() }];
            } 
            else if (message.type === 'noteOff') {
                stopNote(message.note);
            }
        } catch (e) {
            console.error("Error processing WebSocket message:", e);
        }
    }

    // Send note events over WebSocket
    function sendNoteEvent(type, note, velocity = 0.7) {
        const message = JSON.stringify({
            type: type,
            note: note,
            velocity: velocity,
            timestamp: new Date().getTime()
        });
        
        send_msg(message);
    }

    function initAudio() {
        if (!audioContext) {
            audioContext = new AudioContext();
            masterGainNode = audioContext.createGain();
            masterGainNode.gain.value = 0.5;

            // Piano tone shaping
            const eqLow = audioContext.createBiquadFilter();
            eqLow.type = "lowshelf";
            eqLow.frequency.value = 250;
            eqLow.gain.value = 3;

            const eqMid = audioContext.createBiquadFilter();
            eqMid.type = "peaking";
            eqMid.frequency.value = 1500;
            eqMid.Q.value = 0.5;
            eqMid.gain.value = -6;

            const eqHigh = audioContext.createBiquadFilter();
            eqHigh.type = "highshelf";
            eqHigh.frequency.value = 4000;
            eqHigh.gain.value = -6;

            // Compression for dynamic control
            const compressor = audioContext.createDynamicsCompressor();
            compressor.threshold.value = -24;
            compressor.knee.value = 30;
            compressor.ratio.value = 12;
            compressor.attack.value = 0.003;
            compressor.release.value = 0.25;

            // Connect the audio processing chain
            masterGainNode.connect(eqLow);
            eqLow.connect(eqMid);
            eqMid.connect(eqHigh);
            eqHigh.connect(compressor);
            compressor.connect(audioContext.destination);
        }
    }

    function createPianoTone(frequency) {
        const oscillators = [];
        const gains = [];

        // Function to create an oscillator with specific settings
        const createOscillator = (type, freqMultiplier, gainValue) => {
            const osc = audioContext.createOscillator();
            const gain = audioContext.createGain();
            osc.type = type;
            osc.frequency.value = frequency * freqMultiplier;
            gain.gain.value = gainValue;
            oscillators.push(osc);
            gains.push(gain);
            return { osc, gain };
        };

        // Fundamental (main note)
        createOscillator('triangle', 1, 0.5);
        
        // Harmonics
        createOscillator('sine', 2, 0.25);    // First overtone
        createOscillator('sine', 3, 0.125);   // Second overtone
        createOscillator('sine', 4, 0.0625);  // Third overtone

        // Slight detuning for richness
        createOscillator('triangle', 1.0003, 0.1);
        createOscillator('triangle', 0.9997, 0.1);

        const finalGain = audioContext.createGain();
        
        // Connect all oscillators through their individual gains to the final gain
        oscillators.forEach((osc, index) => {
            osc.connect(gains[index]);
            gains[index].connect(finalGain);
        });
        
        finalGain.connect(masterGainNode);

        return {
            oscillators,
            gainNode: finalGain
        };
    }

    function createEnvelope(gainNode, velocity = 1) {
        const now = audioContext.currentTime;
        gainNode.gain.cancelScheduledValues(now);
        
        // Piano-like envelope
        gainNode.gain.setValueAtTime(0, now);
        gainNode.gain.linearRampToValueAtTime(velocity, now + 0.005);
        gainNode.gain.linearRampToValueAtTime(velocity * 0.7, now + 0.1);
        gainNode.gain.exponentialRampToValueAtTime(velocity * 0.5, now + 0.3);
        gainNode.gain.exponentialRampToValueAtTime(0.001, now + 3);
    }

    function cleanupNote(note) {
        const noteData = activeNotes.get(note);
        if (noteData) {
            try {
                const { oscillators, gainNode } = noteData;
                oscillators.forEach(osc => {
                    osc.stop();
                    osc.disconnect();
                });
                gainNode.disconnect();
            } catch (e) {
                console.log('Note cleanup error:', e);
            }
            activeNotes.delete(note);
        }
        
        if (cleanupTimers.has(note)) {
            clearTimeout(cleanupTimers.get(note));
            cleanupTimers.delete(note);
        }
    }

    function cleanupAllNotes() {
        cleanupTimers.forEach(timer => clearTimeout(timer));
        cleanupTimers.clear();
        activeNotes.forEach((_, note) => {
            cleanupNote(note);
        });
        activeNotes.clear();
    }

    function stopAllNotes() {
        activeNotes.forEach((_, note) => {
            stopNote(note);
        });
    }

    function startNote(note, velocity = 0.7) {
        const frequency = noteFrequencies.get(note);
        if (!frequency) return;

        if (!audioContext || audioContext.state === 'closed') {
            initAudio();
        }
        
        if (audioContext.state === 'suspended') {
            audioContext.resume();
        }

        cleanupNote(note);

        const { oscillators, gainNode } = createPianoTone(frequency);
        
        try {
            oscillators.forEach(osc => osc.start());
            createEnvelope(gainNode, velocity);

            activeNotes.set(note, {
                oscillators,
                gainNode,
                startTime: audioContext.currentTime
            });

            // Add visual feedback
            const keyElement = document.querySelector(`[data-note="${note}"]`);
            if (keyElement) {
                keyElement.classList.add('active');
            }
        } catch (e) {
            console.log('Start note error:', e);
            cleanupNote(note);
        }
    }

    function stopNote(note) {
        const noteData = activeNotes.get(note);
        if (noteData) {
            try {
                const { gainNode } = noteData;
                const now = audioContext.currentTime;
                gainNode.gain.cancelScheduledValues(now);
                gainNode.gain.setValueAtTime(gainNode.gain.value, now);
                gainNode.gain.exponentialRampToValueAtTime(0.001, now + 0.5);
                
                if (cleanupTimers.has(note)) {
                    clearTimeout(cleanupTimers.get(note));
                }
                
                const cleanupTimer = setTimeout(() => {
                    cleanupNote(note);
                }, 500);
                
                cleanupTimers.set(note, cleanupTimer);

                // Remove visual feedback
                const keyElement = document.querySelector(`[data-note="${note}"]`);
                if (keyElement) {
                    keyElement.classList.remove('active');
                }
            } catch (e) {
                console.log('Stop note error:', e);
                cleanupNote(note);
            }
        }
    }

    function handleKeyDown(event) {
        if (event.repeat) return;
        
        const key = event.key.toLowerCase();
        if (keyboardMap[key]) {
            const note = keyboardMap[key];
            startNote(note);
            
            // Send note over WebSocket
            sendNoteEvent('noteOn', note);
        }
    }

    function handleKeyUp(event) {
        const key = event.key.toLowerCase();
        if (keyboardMap[key]) {
            const note = keyboardMap[key];
            stopNote(note);
            
            // Send note off over WebSocket
            sendNoteEvent('noteOff', note);
        }
    }

    function handleMouseDown(note, event) {
        const velocity = event.touches ? 
            event.touches[0].force || 0.7 : 
            0.7;
            
        startNote(note, velocity);
        
        // Send note over WebSocket
        sendNoteEvent('noteOn', note, velocity);
    }

    function handleMouseUp(note) {
        stopNote(note);
        
        // Send note off over WebSocket
        sendNoteEvent('noteOff', note);
    }

    function handleTouchStart(event, note) {
        event.preventDefault();
        handleMouseDown(note, event);
    }

    function handleTouchEnd(event, note) {
        event.preventDefault();
        handleMouseUp(note);
    }

    // Generate visible keys for two octaves
    const visibleKeys = [];
    ['4', '5'].forEach(octave => {
        ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B'].forEach(note => {
            visibleKeys.push({
                note: note + octave,
                isSharp: note.includes('#')
            });
        });
    });

    // Function to change room
    function changeRoom() {
        connect_ws(roomName, handleWebSocketMessage);
        connected = true;
    }
</script>

<style>
    .piano {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 20px;
        background: #f5f5f5;
        border-radius: 10px;
        box-shadow: 0 4px 8px rgba(0,0,0,0.2);
    }

    .keyboard {
        position: relative;
        display: flex;
        margin: 20px 0;
        background: #333;
        padding: 5px;
        border-radius: 5px;
    }

    .key {
        position: relative;
        display: flex;
        align-items: flex-end;
        justify-content: center;
        padding-bottom: 8px;
        font-size: 12px;
        font-family: monospace;
        user-select: none;
        cursor: pointer;
    }

    .white-key {
        width: 40px;
        height: 150px;
        background: white;
        border: 1px solid #ccc;
        border-radius: 0 0 4px 4px;
        z-index: 1;
    }

    .black-key {
        width: 24px;
        height: 90px;
        background: #333;
        margin: 0 -12px;
        z-index: 2;
        border-radius: 0 0 3px 3px;
        color: white;
    }

    .key.active {
        background: #e0e0e0;
    }

    .black-key.active {
        background: #666;
    }

    .title {
        font-size: 24px;
        color: #333;
        margin-bottom: 20px;
        font-family: Arial, sans-serif;
    }

    .controls {
        margin-bottom: 20px;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 10px;
    }

    .key-hint {
        position: absolute;
        bottom: 2px;
        font-size: 10px;
        color: #666;
    }

    .black-key .key-hint {
        color: #ccc;
    }

    .room-controls {
        display: flex;
        gap: 10px;
        margin-bottom: 15px;
    }

    .room-controls input {
        padding: 8px;
        border: 1px solid #ccc;
        border-radius: 4px;
    }

    .room-controls button {
        padding: 8px 16px;
        background: #4c63af;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }

    .room-controls button:hover {
        background: #3a4f8a;
    }

    .connection-status {
        font-size: 14px;
        margin-bottom: 10px;
    }

    .connection-status.connected {
        color: green;
    }

    .connection-status.disconnected {
        color: red;
    }

    .received-notes {
        margin-top: 20px;
        padding: 10px;
        border: 1px solid #ddd;
        border-radius: 5px;
        width: 100%;
        max-width: 400px;
        max-height: 100px;
        overflow-y: auto;
        font-size: 12px;
    }

    .received-note {
        margin: 2px 0;
        padding: 2px 5px;
        background: #eef;
        border-radius: 3px;
    }
</style>

<div class="piano">
    <h1 class="title">Collaborative Virtual Piano</h1>
    
    <div class="controls">
        <div class="connection-status {connected ? 'connected' : 'disconnected'}">
            {connected ? '🟢 Connected to WebSocket' : '🔴 Disconnected'}
        </div>
        
        <div class="room-controls">
            <input type="text" bind:value={roomName} placeholder="Enter room name" />
            <button on:click={changeRoom}>Join Room</button>
        </div>
    </div>
    
    <div class="keyboard">
        {#each visibleKeys as {note, isSharp}}
            <div 
                class="key {isSharp ? 'black-key' : 'white-key'}"
                data-note={note}
                on:mousedown={(e) => handleMouseDown(note, e)}
                on:mouseup={() => handleMouseUp(note)}
                on:mouseleave={() => handleMouseUp(note)}
                on:touchstart={(e) => handleTouchStart(e, note)}
                on:touchend={(e) => handleTouchEnd(e, note)}
            >
                {#each Object.entries(keyboardMap) as [key, mappedNote]}
                    {#if mappedNote === note}
                        <span class="key-hint">{key}</span>
                    {/if}
                {/each}
            </div>
        {/each}
    </div>
    
    {#if receivedNotes.length > 0}
        <div class="received-notes">
            <h3>Recent Received Notes</h3>
            {#each receivedNotes.slice(-10) as note}
                <div class="received-note">
                    {note.note} at {note.timestamp.toLocaleTimeString()}
                </div>
            {/each}
        </div>
    {/if}
</div>