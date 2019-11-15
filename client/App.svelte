<script>
import { onMount } from 'svelte';
import QR from './QR.svelte'

var params = null
var login = null
var withdraw = null
var pay = null

onMount(async () => {
  let r = await fetch('/get-params')
  params = await r.json()

  var es = new EventSource('/user-data?session=' + params.session)
  es.addEventListener('login', e => { login = JSON.parse(e.data) })
  es.addEventListener('withdraw', e => { withdraw = JSON.parse(e.data) })
  es.addEventListener('pay', e => { pay = JSON.parse(e.data) })
})
</script>

<style>
  #main {
    margin: 23px auto;
    width: 1200px;
    max-width: 100%;
  }
  h1 {
    text-align: center;
  }
  main {
    display: flex;
    justify-content: space-between;
  }
  main > * {
    margin: 12px;
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 300px;
  }
  th {
    padding-right: 20px;
  }
  td {
    white-space: pre-wrap;
    word-break: break-all;
  }
</style>

<div id="main">
  <h1>lnurl playground</h1>
  <main>
  {#if params}
    <div>
      <a href="lightning:{params.lnurlpay}"><QR value={params.lnurlpay} color="#000" /></a>
      <code>lnurl-pay</code>
      {#if pay}
        <h4>Values received from wallet:</h4>
        <table>
          <tr>
            <th>amount</th>
            <td><code>{pay.amount}</code></td>
          </tr>
          <tr>
            <th>fromnodes</th>
            <td><code>{pay.fromnodes}</code></td>
          </tr>
        </table>
      {/if}
    </div>
    <div>
      <a href="lightning:{params.lnurlwithdraw}"><QR value={params.lnurlwithdraw} color="#000" /></a>
      <code>lnurl-withdraw</code>
      {#if withdraw && withdraw.invoice}
        <h4>Values received from wallet:</h4>
        <table>
          <tr>
            <th>pr</th>
            <td><code>{withdraw.invoice}</code></td>
          </tr>
          <tr>
            <th>k1</th>
            <td><code>{withdraw.k1}</code></td>
          </tr>
        </table>
      {/if}
    </div>
    <div>
      <a href="lightning:{params.lnurllogin}"><QR value={params.lnurllogin} color="#000" /></a>
      <code>lnurl-auth</code>
      {#if login}
        <h4>Values received from wallet:</h4>
        <table>
          <tr>
            <th>key</th>
            <td><code>{login.key}</code></td>
          </tr>
          <tr>
            <th>k1</th>
            <td><code>{login.k1}</code></td>
          </tr>
          <tr>
            <th>sig</th>
            <td><code>{login.sig}</code></td>
          </tr>
        </table>
      {/if}
    </div>
  {/if}
  </main>
</div>
