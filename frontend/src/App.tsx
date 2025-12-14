import { useState, useEffect } from 'react'
import axios from 'axios'

// Configure axios base URL
const api = axios.create({
  baseURL: 'http://localhost:8080'
})

interface UTXO {
  txid: string;
  vout: number;
  address: string;
  amount: number;
  confirmations: number;
}

function App() {
  const [balance, setBalance] = useState<number | null>(null)
  const [address, setAddress] = useState<string>('')
  const [utxos, setUtxos] = useState<UTXO[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')

  const fetchBalance = async () => {
    try {
      const res = await api.get('/balance')
      setBalance(res.data.balance)
    } catch (err: any) {
      console.error(err)
      setError('Failed to fetch balance')
    }
  }

  const fetchUTXOs = async () => {
    try {
      const res = await api.get('/utxos')
      setUtxos(res.data.utxos)
    } catch (err: any) {
      console.error(err)
      setError('Failed to fetch UTXOs')
    }
  }

  const getNewAddress = async () => {
    setLoading(true)
    try {
      const res = await api.get('/address')
      setAddress(res.data.address)
      // Refresh list as we added an address? No, address doesn't affect list until funds received.
    } catch (err: any) {
      console.error(err)
      setError('Failed to get new address')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchBalance()
    fetchUTXOs()
    const interval = setInterval(() => {
      fetchBalance()
      fetchUTXOs()
    }, 10000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div style={{ padding: '20px', fontFamily: 'Arial, sans-serif' }}>
      <h1>Bitcoin Wallet</h1>
      {error && <div style={{ color: 'red' }}>{error}</div>}
      
      <div style={{ marginBottom: '20px', padding: '10px', border: '1px solid #ccc' }}>
        <h2>Balance</h2>
        <p style={{ fontSize: '24px', fontWeight: 'bold' }}>
          {balance !== null ? `${balance} BTC` : 'Loading...'}
        </p>
        <button onClick={fetchBalance}>Refresh Balance</button>
      </div>

      <div style={{ marginBottom: '20px', padding: '10px', border: '1px solid #ccc' }}>
        <h2>Receive Bitcoin</h2>
        <button onClick={getNewAddress} disabled={loading}>
          {loading ? 'Generating...' : 'Get New Address'}
        </button>
        {address && (
          <div style={{ marginTop: '10px' }}>
            <strong>Address:</strong> <code>{address}</code>
          </div>
        )}
      </div>

      <div style={{ padding: '10px', border: '1px solid #ccc' }}>
        <h2>UTXOs</h2>
        <button onClick={fetchUTXOs}>Refresh UTXOs</button>
        {utxos.length === 0 ? (
          <p>No unspent outputs found.</p>
        ) : (
          <table style={{ width: '100%', marginTop: '10px', borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ textAlign: 'left', borderBottom: '1px solid #eee' }}>
                <th>TxID</th>
                <th>Vout</th>
                <th>Address</th>
                <th>Amount</th>
                <th>Confirmations</th>
              </tr>
            </thead>
            <tbody>
              {utxos.map((utxo) => (
                <tr key={`${utxo.txid}-${utxo.vout}`} style={{ borderBottom: '1px solid #eee' }}>
                  <td>{utxo.txid.substring(0, 10)}...</td>
                  <td>{utxo.vout}</td>
                  <td>{utxo.address}</td>
                  <td>{utxo.amount}</td>
                  <td>{utxo.confirmations}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

export default App

