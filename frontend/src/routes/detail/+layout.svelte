<script>

    let {children} = $props();

</script>

<main id="detail-box">
    {@render children()}
</main>

<style>
    #detail-box {
        backdrop-filter: blur(10px);
        background-color: rgb(218 227 255 / 0.4);
        height: 100vh;
        border-radius: 6px;
        overflow: hidden;
    }
</style>

