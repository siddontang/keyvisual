/*
  change your API host here
  we use convert python third-party module to convert our array-like data to 
  matrix which is needed for heatmap visualizer
*/

const tickDataAPIPrefix = '/heatmaps?start=-60m&tag='
var rawInfo,
  allRanges = []
var heatmapType = 'written_bytes'

let ctime = 0,
  switches = {}

/* 
  leave it alone, or set it up by yourself
  install the requirements.txt by pip, and run `python server`
*/

function getData(type) {
  return fetch(tickDataAPIPrefix + type)
    .then(res => res.json())
    .then(json => {
      rawInfo = json
      try {
        const tl =
          // '\t' +
          // '\t\t\t' +
          json.heatmaps[0].values[0].map((i, idx) => {
            return idx + 'm'
            // return 'time-' + idx
          })
        // .join('\t')

        let dlines = [],
          b_num = 0
        json.heatmaps.forEach(h => {
          h.ranges.forEach((i, idx) => {
            const ds = h.labels[2] ? 'index ' + h.labels[2] : 'Data'
            dlines.push(
              [
                `Bucket: bucket-${b_num}`,
                'DB: ' + h.labels[0],
                'Table: ' + h.labels[1],
                'Data Type: ' + ds, //? h.labels[2] : 'DATA',
                ...h.values[idx].map(i => i + 1)
              ]
              // .join('\t')
            )
            b_num += 1
          })
        })

        // return tl + '\n' + dlines.join('\n')
        return [tl, ...dlines]
      } catch (e) {}
    })
}

var about_string = ''

function getClusterColors(dlines) {
  const all_colors = [
    '#393b79',
    '#aec7e8',
    '#ff7f0e',
    '#ffbb78',
    '#98df8a',
    '#bcbd22',
    '#404040',
    '#ff9896',
    '#c5b0d5',
    '#8c564b',
    '#1f77b4',
    '#5254a3',
    '#FFDB58',
    '#c49c94',
    '#e377c2',
    '#7f7f7f',
    '#2ca02c',
    '#9467bd',
    '#dbdb8d',
    '#17becf',
    '#637939',
    '#6b6ecf',
    '#9c9ede',
    '#d62728',
    '#8ca252',
    '#8c6d31',
    '#bd9e39',
    '#e7cb94',
    '#843c39',
    '#ad494a',
    '#d6616b',
    '#7b4173',
    '#a55194',
    '#ce6dbd',
    '#de9ed6'
  ]

  return [1, 2, 3].map(i => {
    const dict = {}
    const s = new Set(dlines.map(l => l[i]))
    // random way to get color, ignore this
    Array.from(s).forEach((n, idx) => {
      dict[n] = all_colors[(idx + 1) * i]
      if (!dict[n]) {
        dict[n] = all_colors[4 + idx]
      }
    })
    return dict
  })
}

function migrateCluster(data) {
  const tl = data[0]
  const dlines = data.slice(1)
  // unpack the matrix

  const catColors = getClusterColors(dlines)
  const cat_colors = {
    col: {},
    row: {
      ['cat-0']: catColors[0],
      ['cat-1']: catColors[1],
      ['cat-2']: catColors[2]
    }
  }

  const col_nodes = tl.map((i, idx) => {
    return {
      clust: tl.length - idx,
      col_index: idx,
      ini: tl.length,
      name: i,
      rank: 0,
      rankvar: 0
    }
  })

  const mat = dlines.map(l => l.slice(4))

  const row_nodes = dlines.map((l, idx) => {
    return {
      name: l[0],
      ['cat-0']: l[1],
      ['cat-1']: l[2],
      ['cat-2']: l[3],
      clust: dlines.length - idx,
      ini: dlines.length - idx
    }
  })

  return { cat_colors, col_nodes, links: [], mat, row_nodes, views: [] }
}

function buildHeatmap(json) {
  console.log(json)

  let network_data = json

  var args = {
    opacity_scale: 'log',
    root: '#container-id-1',
    network_data: network_data,
    about: about_string,
    sidebar_width: 0,
    use_sidebar: false,
    // 'ini_view':{'N_row_var':20}
    ini_expand: true,
    make_row_tooltip_handler: d => {
      const x = rawInfo[0]['buckets'][d.rank]
      return `from ${x['start']} to ${x['end']}`
    },
    make_col_tooltip_handler: d => {
      return ''
      return rawInfo[d.rank].time
    },
    matrix_tip_str_handler: (d, inst_value) => {
      // const len = rawInfo.heatmaps[0].values.length
      // const hidx = Math.floor(d.pos_y / len)
      // const valIdx = d.pos_y % len
      // const x = rawInfo.heatmaps[hidx].ranges[valIdx]
      const x = allRanges[d.pos_y]

      // const x = rawInfo[0]['buckets'][d.pos_y]
      const row_name = `key range from ${JSON.stringify(
        x['start']
      )} to ${JSON.stringify(x['end'])}`
      const col_name = '' // rawInfo[d.pos_x].time
      tooltip_string =
        '<p>' +
        row_name +
        ' and at time ' +
        col_name +
        '</p>' +
        '<div> value: ' +
        inst_value +
        '</div>'
      return tooltip_string
    }
  }
  allRanges = []
  rawInfo.heatmaps.forEach(h => {
    allRanges = [...allRanges, ...h.ranges]
  })

  resize_container(args)

  d3.select(window).on('resize', function() {
    resize_container(args)
    cgm.resize_viz()
  })

  cgm = Heatmap(args)

  // check_setup_enrichr(cgm)

  d3.select(cgm.params.root + ' .wait_message').remove()

  $.busyLoadFull('hide')

  $('.wait_message').html('Heatmap for ' + heatmapType)
}

function make_clust(type) {
  $.busyLoadFull('show')

  getData(type).then(data => {
    const json = migrateCluster(data)
    return buildHeatmap(json)
  })

  // d3.json('json/' + inst_network, function(network_data) {
  //   // define arguments object

  // })
}

function resize_container(args) {
  var screen_width = window.innerWidth
  var screen_height = window.innerHeight - 20

  d3.select(args.root)
    .style('width', screen_width + 'px')
    .style('height', screen_height + 'px')
}

make_clust(heatmapType)

$('#x-switch-menu').click(e => {
  // $('#svg_container-id-1').remove()
  // $('#svg_container-id-1').busyLoad('show')
  //   const canvas = $('.')
  //   const context = canvas.getContext('2d');

  // context.clearRect(0, 0, canvas.width, canvas.height);
  const t = $(e.target.parentElement).data('fetch-label')
  heatmapType = t
  make_clust(t)
})
