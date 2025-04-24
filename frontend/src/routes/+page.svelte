<script>
	import { connect_ws, send_msg } from "$lib/ws.svelte.js";

	let msg = "";
	let messages = [];
	let audio;
	let connected = false;

	function connect() {
		if (!connected) {
			connect_ws("grup1", (data) => {
				messages = [...messages, data];
				if (data === "play") {
					if (audio) {
						audio.currentTime = 0;
						audio.play();
                        console.log("played")
					}
				}
			});
			connected = true;
		}
	}

	function chat() {
		send_msg(msg);
		msg = "";
	}
</script>

<audio bind:this={audio} src="http://127.0.0.1:3000/files/audio.mp3" preload="auto"></audio>

<button on:click={connect}>connect to websocket</button>

<div>
	<input type="text" bind:value={msg} />
	<button on:click={chat}>send message</button>
</div>

{#each messages as msg}
	<div>{msg}</div>
{/each}
