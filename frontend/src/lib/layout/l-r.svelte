<script>
    import "@wailsio/runtime"

    let {left, right} = $props()
</script>

<div class="l-r-box">
    <div class="l">
        {@render left?.()}
    </div>

    <div class="r">
        {@render right?.()}
    </div>
</div>

<style>
    .l-r-box {
        display: grid;
        grid-template-columns: 1fr 50px;
        grid-gap: 0 7px;
        height: 100%;
        padding: 5px;

        .l {
            background-color: #fff;
            border-radius: 4px;
        }

        .r {
            --wails-draggable: drag;
            user-select: none;
            background-color: #fff;
            border-radius: 4px;

        }
    }
</style>