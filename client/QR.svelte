<script>
import { onMount } from 'svelte';
import QRious from 'qrious'

export var value
export var color

let id = parseInt(Math.random() * 100000)

onMount(async () => {
  new QRious({
    element: document.getElementById('c-' + id),
    value: value,
    backgroundAlpha: 0,
    foreground: color,
    size: 300
  })
})
</script>

<canvas id="c-{id}"></canvas>
