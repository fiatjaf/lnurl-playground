<script>
  import {onMount, afterUpdate} from 'svelte'
  import QR from './QR.svelte'

  const subProtocolColor = {
    login: '#82f1ff',
    pay: '#95f0a4',
    withdraw: '#ff9469',
    channel: '#a695a9',
    null: 'transparent'
  }

  const hex = '0123456789abcdef'
  var session = ''
  for (let i = 0; i < 64; ++i) {
    session += hex.charAt(Math.floor(Math.random() * hex.length))
  }

  var params = null
  var lastEventKind = null
  var login = null
  var withdraw_req = null
  var withdraw = null
  var pay_req = null
  var pay = null
  var pay_result = null
  var channel_req = null
  var channel = null

  // preferences
  var disposable = true
  var metadataSize = 23

  afterUpdate(() => {
    document.body.style.borderColor = subProtocolColor[lastEventKind]
  })

  onMount(async () => {
    var es = new EventSource(`/session?session=${session}`)
    es.addEventListener('params', e => {
      params = JSON.parse(e.data)
    })
    es.addEventListener('login', e => {
      login = JSON.parse(e.data)
      lastEventKind = 'login'
    })
    es.addEventListener('withdraw-req', e => {
      withdraw_req = JSON.parse(e.data)
      withdraw = null
      lastEventKind = 'withdraw'
    })
    es.addEventListener('withdraw', e => {
      withdraw = JSON.parse(e.data)
      lastEventKind = 'withdraw'
    })
    es.addEventListener('pay-req', e => {
      pay_req = JSON.parse(e.data)
      pay = null
      pay_result = null
      lastEventKind = 'pay'
    })
    es.addEventListener('pay', e => {
      pay = JSON.parse(e.data)
      lastEventKind = 'pay'
    })
    es.addEventListener('pay_result', e => {
      pay_req = null
      pay_result = JSON.parse(e.data)
      lastEventKind = 'pay'
    })
    es.addEventListener('channel-req', e => {
      channel_req = JSON.parse(e.data)
      channel = null
      lastEventKind = 'channel'
    })
    es.addEventListener('channel', e => {
      channel = JSON.parse(e.data)
      lastEventKind = 'channel'
    })
  })

  function setPreferences(e) {
    e.preventDefault()
    fetch(`/set-preferences?session=${session}`, {
      method: 'post',
      body: `disposable=${disposable}`,
      headers: {'Content-Type': 'application/x-www-form-urlencoded'}
    })
  }

  function triggerNotify(e) {
    e.preventDefault()
    fetch(`/trigger-notify`, {
      method: 'post',
      body: `notifyURL=${encodeURIComponent(withdraw.balanceNotify)}`,
      headers: {'Content-Type': 'application/x-www-form-urlencoded'}
    })
  }

  function tryPrettyJSON(jsonstr) {
    var json

    if (typeof jsonstr === 'object') {
      json = jsonstr
    } else {
      try {
        json = JSON.parse(jsonstr)
      } catch (_) {
        return `invalid JSON: ${jsonstr}`
      }
    }

    return JSON.stringify(json, null, 2)
  }
</script>

<div id="main">
  <header>
    <h1 on:click={() => location.reload()}>lnurl playground</h1>
    <small>
      <a href="/codec">lnurl encoder/decoder</a>
    </small>
  </header>
  <main>
    {#if params}
      <div class:hidden={lastEventKind && lastEventKind !== 'pay'}>
        <a href="lightning:{params.lnurlpay}"
          ><QR value={params.lnurlpay} color="#000" /></a
        >
        <code>lnurl-pay</code>

        {#if pay_req}
          <h4>Params sent to wallet:</h4>
          <table>
            <tr>
              <th>tag</th>
              <td><code>{pay_req.tag}</code></td>
            </tr>
            <tr>
              <th>callback</th>
              <td><code>{pay_req.callback}</code></td>
            </tr>
            <tr>
              <th>minSendable / maxSendable</th>
              <td>
                <code>{pay_req.minSendable} / {pay_req.maxSendable}</code>
              </td>
            </tr>
            <tr>
              <th>metadata</th>
              <td>
                <pre>
              <code>{tryPrettyJSON(pay_req.metadata)}</code>
            </pre>
              </td>
            </tr>
          </table>
        {/if}

        <!---->

        {#if pay}
          <h4>Values received from wallet:</h4>
          <table>
            <tr>
              <th>amount</th>
              <td><code>{pay.amount}</code></td>
            </tr>
            {#if pay.comment}
              <tr>
                <th>comment</th>
                <td><code>{pay.comment}</code></td>
              </tr>
            {/if}
            {#if pay.payerdata}
              <tr>
                <th>payerdata</th>
                <td><code>{tryPrettyJSON(pay.payerdata)}</code></td>
              </tr>
            {/if}
          </table>
          {#if pay_result}
            <h4>Final values sent to wallet:</h4>
            <table>
              <tr>
                <th>pr</th>
                <td><code>{pay_result.pr}</code></td>
              </tr>
              <tr>
                <th>successAction</th>
                <td><code>{JSON.stringify(pay_result.successAction)}</code></td>
              </tr>
            </table>
          {/if}
        {/if}
      </div>
      <div class:hidden={lastEventKind && lastEventKind !== 'withdraw'}>
        <a href="lightning:{params.lnurlwithdraw}"
          ><QR value={params.lnurlwithdraw} color="#000" /></a
        >
        <code>lnurl-withdraw</code>
        {#if withdraw_req}
          <h4>Params sent to wallet:</h4>
          <table>
            <tr>
              <th>tag</th>
              <td><code>{withdraw_req.tag}</code></td>
            </tr>
            <tr>
              <th>callback</th>
              <td><code>{withdraw_req.callback}</code></td>
            </tr>
            <tr>
              <th>k1</th>
              <td><code>{withdraw_req.k1}</code></td>
            </tr>
            <tr>
              <th>minWithdrawable / maxWithdrawable</th>
              <td>
                <code
                  >{withdraw_req.minWithdrawable} /
                  {withdraw_req.maxWithdrawable}</code
                >
              </td>
            </tr>
            <tr>
              <th>defaultDescription</th>
              <td><code>{withdraw_req.defaultDescription}</code></td>
            </tr>
            <tr>
              <th>balanceCheck</th>
              <td><code>{withdraw_req.balanceCheck}</code></td>
            </tr>
            <tr>
              <th>payLink</th>
              <td><code>{withdraw_req.payLink}</code></td>
            </tr>
          </table>
        {/if}

        <!---->

        {#if withdraw}
          <h4>Values received from wallet:</h4>
          <table>
            <tr>
              <th>pr</th>
              <td><code>{withdraw.pr}</code></td>
            </tr>
            <tr>
              <th>k1</th>
              <td><code>{withdraw.k1}</code></td>
            </tr>
            <tr>
              <th>balanceNotify</th>
              <td>
                <code>{withdraw.balanceNotify}</code>
                {#if withdraw.balanceNotify}
                  <button on:click={triggerNotify}>trigger</button>
                {/if}
              </td>
            </tr>
          </table>
        {/if}
      </div>

      <div class:hidden={lastEventKind && lastEventKind !== 'login'}>
        <a href="lightning:{params.lnurllogin}"
          ><QR value={params.lnurllogin} color="#000" /></a
        >
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
      <div class:hidden={lastEventKind && lastEventKind !== 'channel'}>
        <a href="lightning:{params.lnurlchannel}"
          ><QR value={params.lnurlchannel} color="#000" /></a
        >
        <code>lnurl-channel</code>

        {#if channel_req}
          <h4>Params sent to wallet:</h4>
          <table>
            <tr>
              <th>tag</th>
              <td><code>{channel_req.tag}</code></td>
            </tr>
            <tr>
              <th>callback</th>
              <td><code>{channel_req.callback}</code></td>
            </tr>
            <tr>
              <th>k1</th>
              <td><code>{channel_req.k1}</code></td>
            </tr>
            <tr>
              <th>uri</th>
              <td><code>{channel_req.uri}</code></td>
            </tr>
          </table>
        {/if}

        <!---->

        {#if channel}
          <h4>Values received from wallet:</h4>
          <table>
            <tr>
              <th>k1</th>
              <td><code>{channel.k1}</code></td>
            </tr>
            <tr>
              <th>remoteid</th>
              <td><code>{channel.remoteid}</code></td>
            </tr>
            <tr>
              <th>private</th>
              <td><code>{channel.private}</code></td>
            </tr>
          </table>
        {/if}
      </div>
      <div id="preferences">
        <form on:submit={setPreferences}>
          <label
            >disposable? <input
              type="checkbox"
              bind:checked={disposable}
            /></label
          >
          <button>set</button>
        </form>
      </div>
    {/if}
  </main>
</div>

<style>
  #main {
    margin: 23px auto;
    width: 1200px;
    max-width: 100%;
  }
  header {
    display: flex;
  }
  header h1 {
    flex-grow: 3;
    text-align: center;
    cursor: pointer;
  }
  header small {
    display: flex;
    justify-content: center;
    align-items: center;
    width: 50px;
  }
  header small a {
    color: #333;
  }
  header small a:hover {
    text-decoration: none;
  }
  input {
    line-height: 2em;
  }
  main {
    display: flex;
    justify-content: space-between;
    flex-wrap: wrap;
  }
  main > * {
    margin: 36px 12px;
    display: flex;
    flex-direction: column;
    align-items: center;
  }
  th {
    padding-right: 20px;
  }
  th,
  td {
    white-space: pre-wrap;
    word-wrap: break-word;
  }
  td {
    word-break: break-all;
  }
  .hidden {
    display: none !important;
  }
  #preferences {
    width: 300px;
  }
  #preferences > * {
    margin: 10px;
  }
  #preferences form {
    display: block;
  }
  #preferences label {
    display: block;
    margin: 4px 0;
  }
  #preferences button {
    background: #e0e0f0;
    padding: 1px 17px;
  }
</style>
