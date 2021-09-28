import svelte from 'rollup-plugin-svelte'
import resolve from '@rollup/plugin-node-resolve'
import commonjs from '@rollup/plugin-commonjs'
import {terser} from 'rollup-plugin-terser'
import css from 'rollup-plugin-css-only'
import inject from '@rollup/plugin-inject'

const production = !!process.env.PRODUCTION

export default {
  input: 'client/main.js',
  output: {
    sourcemap: true,
    format: 'iife',
    name: 'app',
    file: 'static/bundle.js'
  },
  plugins: [
    svelte({
      compilerOptions: {dev: !production}
    }),

    resolve({
      browser: true,
      dedupe: importee =>
        importee === 'svelte' || importee.startsWith('svelte/'),
      preferBuiltins: false
    }),

    commonjs(),

    inject({
      Buffer: ['buffer', 'Buffer']
    }),

    css({output: 'bundle.css'}),

    // If we're building for production (npm run build
    // instead of npm run dev), minify
    production && terser()
  ],
  watch: {
    clearScreen: false
  }
}
